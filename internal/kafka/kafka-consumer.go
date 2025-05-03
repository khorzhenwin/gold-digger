package kafka

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"github.com/khorzhenwin/gold-digger/internal/config"
	"github.com/khorzhenwin/gold-digger/internal/models"
	"github.com/twmb/franz-go/pkg/sasl/oauth"
	"log"

	"github.com/twmb/franz-go/pkg/kgo"
)

// StartKafkaConsumer consumes messages from a Kafka topic and pushes them into a channel
func StartKafkaConsumer(kafkaCfg *config.KafkaConfig, groupID string, out chan<- models.TickerPrice) {
	log.Println("📥 Starting Kafka consumer...")
	token, err := getOAuthToken(kafkaCfg.ClientId, kafkaCfg.ClientSecret)

	client, err := kgo.NewClient(
		kgo.SeedBrokers(kafkaCfg.Broker),
		kgo.DialTLSConfig(&tls.Config{}),
		kgo.SASL(oauth.Auth{
			Token: token,
		}.AsMechanism()),
		kgo.ConsumerGroup(groupID),
		kgo.ConsumeTopics(kafkaCfg.TickerPriceTopic),
		kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()), // ✅ start from earliest
		//kgo.WithLogger(kgo.BasicLogger(log.Writer(), kgo.LogLevelDebug, nil)),
	)
	if err != nil {
		log.Fatalf("❌ Failed to create Kafka consumer: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	for {
		log.Println("🔄 Polling Kafka for new messages...")
		fetches := client.PollFetches(ctx)

		fetchCount := 0

		fetches.EachPartition(func(p kgo.FetchTopicPartition) {
			log.Printf("📦 Partition %s has %d records", p.Topic, len(p.Records))

			for _, record := range p.Records {
				fetchCount++
				var msg models.TickerPrice
				if err := json.Unmarshal(record.Value, &msg); err != nil {
					log.Printf("⚠️ Failed to parse message: %v", err)
					continue
				}
				out <- msg
			}
		})

		if fetchCount == 0 {
			log.Println("🕸️ No messages received in this poll.")
		}

		if err := fetches.Err(); err != nil {
			log.Printf("⚠️ Fetch error: %v", err)
		}
	}
}
