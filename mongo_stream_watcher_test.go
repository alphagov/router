package main

import (
  "context"
  "fmt"
  "os"
  "testing"
  "time"

  . "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

  "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	mswtDB *mongo.Database
	testContext context.Context = context.Background()
)

type Receiver struct {
  reloadChan  chan bool
  counter     int
}

func createReceiver() *Receiver {
  r := Receiver {
    reloadChan: make(chan bool, 1),
    counter: 0,
  }

  go func() {
    for range r.reloadChan {
      r.counter += 1
    }
  }()

  return &r
}

func databaseUrl() string {
  databaseUrl := os.Getenv("ROUTER_MONGO_URL")

	if databaseUrl == "" {
		databaseUrl = "127.0.0.1"
	}

  return databaseUrl
}

func initDbHelper() error {
	uri := "mongodb://" + databaseUrl()
	client, err := mongo.Connect(testContext, options.Client().ApplyURI(uri))
	if err != nil {
		return fmt.Errorf("Failed to connect to mongo: " + err.Error())
	}
	// sess.SetSyncTimeout(10 * time.Minute)
	// sess.SetSocketTimeout(10 * time.Minute)

	mswtDB = client.Database("mongo_stream_watcher_test")
	return nil
}

func addTestRecord() {
	_, err := mswtDB.Collection("test-data").InsertOne(testContext, bson.M{"test_id": 1, "value": "hello"})
	Expect(err).To(BeNil())
}

func addIgnoreRecord() {
	_, err := mswtDB.Collection("ignore-data").InsertOne(testContext, bson.M{"test_id": 1, "value": "hello"})
	Expect(err).To(BeNil())
}


func TestRouter(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Mongo Stream Watcher Suite")
}

var _ = Describe("Mongo Stream Watcher", func() {
  var (
		receiver  *Receiver
    msw       *MongoStreamWatcher
	)

  BeforeEach(func() {
    initDbHelper()
    addIgnoreRecord()
    receiver = createReceiver()
    msw = &MongoStreamWatcher{
      MongoURL:         databaseUrl(),
      MongoDbName:      "mongo_stream_watcher_test",
      CollectionNames: 	[]string{"test-data"},
      ChangeChan:			  receiver.reloadChan,
    }
    msw.startWatcher()
  })

	Context("When a change occurs in a watched collection", func() {
		It("should signal the reload channel", func() {
      Expect(receiver.counter).To(Equal(0))
      addTestRecord()
      time.Sleep(5 * time.Millisecond)
      Expect(receiver.counter).To(Equal(1))
		})
  })

  Context("When a change occurs in an ignored collection", func() {
		It("should not the reload channel", func() {
      Expect(receiver.counter).To(Equal(0))
      addIgnoreRecord()
      time.Sleep(5 * time.Millisecond)
      Expect(receiver.counter).To(Equal(0))
		})
  })
})
