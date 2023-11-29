package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	"github.com/go-redis/redis/v8"
	"github.com/streadway/amqp"
)

var redisDB *redis.Client

func main() {
	// Create a Redis client
	redisDB = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password
	})

	// Connect to RabbitMQ
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	// Create a RabbitMQ channel
	ch, err := conn.Channel()
	if err != nil {
		log.Fatal(err)
	}
	defer ch.Close()

	// Declare exchange and queue for communication
	exchangeName := "user_exchange"
	queueName := "user_queue"
	if err := declareExchangeAndQueue(ch, exchangeName, queueName); err != nil {
		log.Fatal(err)
	}

	// Consume messages from RabbitMQ
	msgs, err := ch.Consume(
		queueName, // queue
		"",        // consumer
		true,      // auto-ack
		false,     // exclusive
		false,     // no-local
		false,     // no-wait
		nil,       // args
	)
	if err != nil {
		log.Fatal(err)
	}

	// Process messages in a goroutine
	go func() {
		for d := range msgs {
			fmt.Printf("Received a message: %s\n", d.Body)
			ProcessRabbitMQMessage(d.Body)
		}
	}()

	fmt.Println("Waiting for messages. To exit, press CTRL+C")
	select {}
}

func declareExchangeAndQueue(ch *amqp.Channel, exchangeName, queueName string) error {
	// Declare exchange
	err := ch.ExchangeDeclare(
		exchangeName, // name
		"fanout",     // type
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare exchange: %v", err)
	}

	// Declare queue
	_, err = ch.QueueDeclare(
		queueName, // name
		false,     // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %v", err)
	}

	// Bind queue to exchange
	err = ch.QueueBind(
		queueName,    // queue name
		"",           // routing key
		exchangeName, // exchange
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to bind queue to exchange: %v", err)
	}

	return nil
}

func ProcessRabbitMQMessage(body []byte) {
	fmt.Println("processRabbitMQMessage")
	var message map[string]interface{}
	if err := json.Unmarshal(body, &message); err != nil {
		log.Println("Error decoding JSON:", err)
		return
	}

	// Extract action and user data from the message
	action, ok := message["action"].(string)
	if !ok {
		log.Println("Action is not a string")
		return
	}

	userData, ok := message["user"].(map[string]interface{})
	if !ok {
		log.Println("User data is not a map")
		return
	}

	fmt.Println(userData, action)

	// Extract and convert user ID to string
	floatId, ok := userData["ID"].(float64)
	if !ok {
		log.Println("Conversion failed. The value is not a float64.")
		return
	}

	stringId := "user-" + strconv.Itoa(int(floatId))

	// Convert user data to JSON
	usr, err := json.Marshal(userData)
	if err != nil {
		log.Fatal(err)
	}

	// Process based on the action type
	switch action {
	case "create", "update":
		// Handle create/update action and update Redis
		fmt.Println(stringId, action)
		if err := updateRedis(stringId, usr); err != nil {
			log.Println("Failed to update Redis:", err)
		}
	case "delete":
		// Handle delete action and delete from Redis
		fmt.Println(stringId, action)
		if err := deleteFromRedis(stringId); err != nil {
			log.Println("Failed to delete from Redis:", err)
		}
	default:
		log.Println("Unknown action:", action)
	}
}

func updateRedis(key string, value []byte) error {
	// Update Redis
	return redisDB.Set(context.Background(), key, value, 0).Err()
}

func deleteFromRedis(key string) error {
	// Delete from Redis
	return redisDB.Del(context.Background(), key).Err()
}
