package main

import (
	"encoding/json"
	"fmt"
	"github.com/streadway/amqp"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"log"
)

// User represents a user in the application
type User struct {
	gorm.Model
	Name  string
	Email string
}

func main() {
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

	// Connect to the SQLite database
	db, err := gorm.Open(sqlite.Open("db.sqlite"), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}
	// Auto-migrate the User model to create the necessary table
	db.AutoMigrate(&User{})

	// Your application logic here...
	//insertUser(db, "alimhv", "mahmoodvand.ali@gmail.com")
	deleteUser(db, 4)

	// Keep the application running
	// select {}
}

// insertUser creates a new user in the database and returns the created user
func insertUser(db *gorm.DB, name, email string) *User {
	newUser := User{Name: name, Email: email}
	db.Create(&newUser)
	return &newUser
}

// deleteUser deletes a user from the database by ID
func deleteUser(db *gorm.DB, id uint) {
	user := User{}
	user.ID = id
	db.Delete(&user)
}

// AfterCreate Gorm hook to capture create event
func (user *User) AfterCreate(tx *gorm.DB) (err error) {
	fmt.Println("AfterCreate")
	// Publish message to RabbitMQ queue
	message := map[string]interface{}{
		"action": "create",
		"user":   user,
	}
	PublishToRabbitMQ(message)
	return nil
}

// AfterUpdate Gorm hook to capture update event
func (user *User) AfterUpdate(tx *gorm.DB) (err error) {
	fmt.Println("AfterUpdate")
	// Publish message to RabbitMQ queue
	message := map[string]interface{}{
		"action": "update",
		"user":   user,
	}
	PublishToRabbitMQ(message)
	return nil
}

// AfterDelete Gorm hook to capture delete event
func (user *User) AfterDelete(tx *gorm.DB) (err error) {
	fmt.Println("AfterDelete")
	// Publish message to RabbitMQ queue
	message := map[string]interface{}{
		"action": "delete",
		"user":   user,
	}
	PublishToRabbitMQ(message)
	return nil
}

// PublishToRabbitMQ publishes a message to the RabbitMQ exchange
func PublishToRabbitMQ(message interface{}) {
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

	// Declare the RabbitMQ exchange
	exchangeName := "user_exchange"
	err = ch.ExchangeDeclare(
		exchangeName, // name
		"fanout",     // type
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	)
	if err != nil {
		log.Fatal(err)
	}

	// Convert the message to JSON
	body, err := json.Marshal(message)
	if err != nil {
		log.Fatal(err)
	}

	// Publish the message to RabbitMQ
	err = ch.Publish(
		exchangeName, // exchange
		"",           // routing key
		false,        // mandatory
		false,        // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        body,
		})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Published message for user %s to RabbitMQ\n", message)
}
