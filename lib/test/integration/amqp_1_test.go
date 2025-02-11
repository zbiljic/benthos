package integration

import (
	"fmt"
	"testing"
	"time"

	"github.com/Azure/go-amqp"
	"github.com/Jeffail/benthos/v3/internal/integration"
	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ = registerIntegrationTest("amqp_1", func(t *testing.T) {
	t.Parallel()

	pool, err := dockertest.NewPool("")
	require.NoError(t, err)

	pool.MaxWait = time.Second * 30
	resource, err := pool.Run("rmohr/activemq", "latest", nil)
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, pool.Purge(resource))
	})

	resource.Expire(900)
	require.NoError(t, pool.Retry(func() error {
		client, err := amqp.Dial(fmt.Sprintf("amqp://guest:guest@localhost:%v/", resource.GetPort("5672/tcp")))
		if err == nil {
			client.Close()
		}
		return err
	}))

	template := `
output:
  amqp_1:
    url: amqp://guest:guest@localhost:$PORT/
    target_address: "queue:/$ID"
    max_in_flight: $MAX_IN_FLIGHT
    metadata:
      exclude_prefixes: [ $OUTPUT_META_EXCLUDE_PREFIX ]

input:
  amqp_1:
    url: amqp://guest:guest@localhost:$PORT/
    source_address: "queue:/$ID"
`
	suite := integration.StreamTests(
		integration.StreamTestOpenClose(),
		integration.StreamTestSendBatch(10),
		integration.StreamTestStreamSequential(1000),
		integration.StreamTestStreamParallel(1000),
		integration.StreamTestStreamParallelLossy(1000),
		integration.StreamTestStreamParallelLossyThroughReconnect(1000),
		integration.StreamTestMetadata(),
		integration.StreamTestMetadataFilter(),
	)
	suite.Run(
		t, template,
		integration.StreamTestOptSleepAfterInput(100*time.Millisecond),
		integration.StreamTestOptSleepAfterOutput(100*time.Millisecond),
		integration.StreamTestOptPort(resource.GetPort("5672/tcp")),
	)
	t.Run("with max in flight", func(t *testing.T) {
		t.Parallel()
		suite.Run(
			t, template,
			integration.StreamTestOptSleepAfterInput(100*time.Millisecond),
			integration.StreamTestOptSleepAfterOutput(100*time.Millisecond),
			integration.StreamTestOptPort(resource.GetPort("5672/tcp")),
			integration.StreamTestOptMaxInFlight(10),
		)
	})
})
