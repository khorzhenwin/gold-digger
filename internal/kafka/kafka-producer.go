package kafka

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/khorzhenwin/gold-digger/internal/config"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/sasl"
	"github.com/twmb/franz-go/pkg/sasl/oauth"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

var (
	client      *kgo.Client
	clientError error
	ready       = make(chan struct{}) // 🔒 block until initialized
)

type oauthBearer struct {
	token string
}

func (o oauthBearer) Name() string {
	return "OAUTHBEARER"
}

func (o oauthBearer) Authenticate(ctx context.Context, host string) (sasl.Session, []byte, error) {
	authBytes := fmt.Sprintf("n,,\x01auth=Bearer %s\x01\x01", o.token)
	return nil, []byte(authBytes), nil
}

func getOAuthToken(clientID, clientSecret string) (string, error) {
	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)

	tokenURL := "https://auth.prd.cloud.redpanda.com/oauth/token"

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var res struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}

	return res.AccessToken, nil
}

func InitKafkaProducer(kafkaConfig *config.KafkaConfig) {
	broker := kafkaConfig.Broker
	//username := kafkaConfig.Username
	//password := kafkaConfig.Password
	clientId := kafkaConfig.ClientId
	clientSecret := kafkaConfig.ClientSecret

	token, err := getOAuthToken(clientId, clientSecret)
	if err != nil {
		log.Fatalf("❌ Failed to get OAuth token: %v", err)
	}

	client, clientError = kgo.NewClient(
		kgo.SeedBrokers(broker),
		kgo.DialTLSConfig(&tls.Config{}),
		kgo.SASL(oauth.Auth{
			Token: token,
		}.AsMechanism()),
	)

	if clientError != nil {
		log.Fatalf("❌ Failed to initialize franz Kafka client: %v", clientError)
	}

	log.Println("🚀 Franz Kafka producer initialized")
	close(ready)
}

func CloseKafkaProducer() {
	if client != nil {
		client.Close()
		log.Println("👋 Closed Franz Kafka client")
	}
}

func PushToKafkaTopic[T any](topic string, data T, key string) {
	<-ready // ⏳ block until Kafka is ready
	if client == nil {
		log.Fatal("💥 Kafka client is nil after supposed init")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	value, err := json.Marshal(data)
	if err != nil {
		log.Println("❌ JSON encode error: %w", err)
	}

	record := &kgo.Record{
		Topic: topic,
		Key:   []byte(key),
		Value: value,
	}

	var wg sync.WaitGroup
	wg.Add(1)

	client.Produce(ctx, record, func(r *kgo.Record, err error) {
		defer wg.Done()
		if err != nil {
			log.Printf("❌ Failed to produce message: %v", err)
		} else {
			log.Printf("✅ Kafka message sent (offset=%d): %s", r.Offset, key)
		}
	})

	wg.Wait()
}
