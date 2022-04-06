package main

type MongoStreamWatcher struct {
  Client          mongo.Client
  ChangeChan      chan bool
  ChannelNames    []String

  invalidateChan  chan bool
  changeStreams   []ChangeStreamInfo
}

type ChangeStreamInfo struct {
	collectionName  string
	stream 					*mongo.ChangeStream
	valid  					bool
}

func (msw *MongoStreamWatcher) startWatcher() {
  collectionCount := len(msw.ChannelNames)
  msw.invalidateChan = make(chan bool, 1)
  msw.changeStreams = [collectionCount]ChangeStreamInfo

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
        msw.tryStreamWatch();
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

func watchChangeStream(context context.Context, cs *mongo.ChangeStream, name string, reloadChan chan bool, invalidateChan chan bool) {
	logInfo(fmt.sprintf("mongo change stream: Listening on %s change stream", name))
	for cs.Next(context) {
		logInfo(fmt.sprintf("mongo change stream: Detected updates in %s collection, signalling reload channel", name))
		reloadChan <- true;
	}

	invalidateChan <- true;
}

func watchChangeStreamInvalidation(msw *MongoStreamWatcher) {
  	for range msw.invalidateChan {
      logInfo("mongo change stream: change stream invalidated, restarting change stream watcher")
      msw.startChangeStreams()
    }
}

func addStreamWatchers(string []collectionNames, changeChan chan bool) {
	logInfo("mgo: setting up management of change streams for ", rt.mongoURL)

	uri := "mongodb://" + rt.mongoURL
	client, err := mongo.Connect(rt.mongoContext, options.Client().ApplyURI(uri))
	if err != nil {
		logWarn(fmt.Sprintf("mongo: error connecting to MongoDB, skipping change stream checking (error: %v)", err))
		return
	}

	// defer client.Disconnect(rt.mongoContext)

	// Streams, so simple but so horrific - we need to poll their setup at first because we
	// can't assume the database is ready for them.

	for (rt.changeStreams[0].isValid == false || rt.changeStreams[1].isValid == false) {
		for i := range rt.changeStreams {
			if !rt.changeStreams[i].isValid {
				rt.tryStreamWatch(client, rt.mongoDbName, &rt.changeStreams[i])
				if rt.changeStreams[i].isValid {
					go iterateStreamCursor(rt.mongoContext, rt.changeStreams[i].stream, rt.ReloadChan, rt.changeStreams[i].invalidChan)
				}
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
}



func (rt *Router) tryStreamWatch(client *mongo.Client, dbName string, changeStream *ChangeStreamInfo) {
	db := client.Database(dbName)
	collection := db.Collection(changeStream.collectionName)

	logInfo(fmt.Sprintf("Connecting to change stream on collection: %s", changeStream.collectionName))
	cs, err := collection.Watch(rt.mongoContext, mongo.Pipeline{})
	if err != nil {
		changeStream.isValid = false
		logWarn("Unable to listen to change stream")
		logWarn(err)
		return
	}

	changeStream.isValid = true
	changeStream.stream = cs
}
