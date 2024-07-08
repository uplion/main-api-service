package main

import (
	"github.com/apache/pulsar-client-go/pulsar"
	"log"
	"os"
	"sync"
)

func initPulsarClient() pulsar.Client {
	pulsarURL := os.Getenv("PULSAR_URL")
	if pulsarURL == "" {
		log.Fatalln("PULSAR_URL environment variable is not set.")
	}

	pulsarToken := os.Getenv("PULSAR_TOKEN")
	clientOptions := pulsar.ClientOptions{
		URL: pulsarURL,
	}

	if pulsarToken != "" {
		clientOptions.Authentication = pulsar.NewAuthenticationToken(pulsarToken)
	}

	client, err := pulsar.NewClient(clientOptions)
	if err != nil {
		log.Fatalln("Could not instantiate Pulsar client: ", err)
	}

	log.Println("Pulsar client initialized")

	return client
}

type ProducerCache struct {
	cache  map[string]pulsar.Producer
	mu     sync.RWMutex
	client *pulsar.Client
}

func NewProducerCache(client *pulsar.Client) *ProducerCache {
	return &ProducerCache{
		cache:  make(map[string]pulsar.Producer),
		client: client,
	}
}

func (p *ProducerCache) GetProducer(topic string) (pulsar.Producer, error) {
	p.mu.RLock()
	prod, ok := p.cache[topic]
	p.mu.RUnlock()
	if ok {
		return prod, nil
	}

	producer, err := (*p.client).CreateProducer(pulsar.ProducerOptions{
		Topic: topic,
	})
	if err != nil {
		return nil, err
	}

	p.mu.Lock()
	p.cache[topic] = producer
	p.mu.Unlock()
	return producer, nil
}
