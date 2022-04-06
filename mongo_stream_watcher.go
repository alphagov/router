package main

import (
  "context"
  "fmt"

  "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoStreamWatcher struct {
  MongoURL        string
  MongoDbName     string
  ChangeChan      chan bool
  CollectionNames []string

  database        *mongo.Database
  invalidateChan  chan bool
  changeStreams   []ChangeStreamInfo
}

type ChangeStreamInfo struct {
	collectionName  string
	stream 					*mongo.ChangeStream
	valid  					bool
}

func (msw *MongoStreamWatcher) startWatcher() {
  msw.invalidateChan = make(chan bool, 1)
  msw.changeStreams = make([]ChangeStreamInfo, len(msw.CollectionNames))

  for i, name := range msw.CollectionNames {
    msw.changeStreams[i] = ChangeStreamInfo {
      collectionName: name,
      stream: nil,
      valid: false,
    }
  }

  logInfo("mongo change stream: initialising change stream watcher manager")

	uri := "mongodb://" + msw.MongoURL
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(uri))
	if err != nil {
		logWarn(fmt.Sprintf("mongo: error connecting to MongoDB, skipping change stream checking (error: %v)", err))
		return
	}
  // defer client.Disconnect(context.Background())

  msw.database = client.Database(msw.MongoDbName)

  logInfo("mongo change stream: starting change stream watchers")
  go watchChangeStreamInvalidation(msw)
  msw.startChangeStreams()
}

func (msw *MongoStreamWatcher) startChangeStreams() {
  if msw.allChangeStreamsValid() {
    return
  }

  for i := range msw.changeStreams {
      if  !msw.changeStreams[i].valid {
        msw.tryStreamWatch(&msw.changeStreams[i]);
        go watchChangeStream(context.Background(), msw.changeStreams[i].stream, msw.changeStreams[i].collectionName, msw.ChangeChan, msw.invalidateChan)
      }
  }
}

func (msw *MongoStreamWatcher) allChangeStreamsValid() bool {
  for i := range msw.changeStreams {
      if  !msw.changeStreams[i].valid {
        return false
      }
  }

  return true
}

func watchChangeStream(context context.Context, cs *mongo.ChangeStream, name string, changeChan chan bool, invalidateChan chan bool) {
	logInfo(fmt.Sprintf("mongo change stream: Listening on %s change stream", name))
	for cs.Next(context) {
		logInfo(fmt.Sprintf("mongo change stream: Detected updates in %s collection, signalling reload channel", name))
		changeChan <- true;
	}

  if err := cs.Err(); err != nil {
		logInfo(err)
	}

	invalidateChan <- true;
}

func watchChangeStreamInvalidation(msw *MongoStreamWatcher) {
  for range msw.invalidateChan {
    logInfo("mongo change stream: change stream invalidated, restarting change stream watcher")
    msw.startChangeStreams()
  }
}

func (msw *MongoStreamWatcher) tryStreamWatch(changeStream *ChangeStreamInfo) {
	collection := msw.database.Collection(changeStream.collectionName)

	logInfo(fmt.Sprintf("mongo change stream: Connecting to change stream on collection: %s", changeStream.collectionName))
	cs, err := collection.Watch(context.Background(), mongo.Pipeline{})
	if err != nil {
		changeStream.valid = false
		logWarn("Unable to listen to change stream")
		logWarn(err)
		return
	}

	changeStream.valid = true
	changeStream.stream = cs
}
