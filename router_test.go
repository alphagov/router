package main

import (
	// "errors"
	"testing"
	// "time"

	"go.mongodb.org/mongo-driver/bson"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type mockMongoDB struct {
	result bson.M
	err    error
}

func (m *mockMongoDB) Run(cmd interface{}, res interface{}) error {
	if m.err != nil {
		return m.err
	} else {
		bytes, err := bson.Marshal(m.result)
		if err != nil {
			return err
		}

		err = bson.Unmarshal(bytes, res)
		if err != nil {
			return err
		}
	}

	return nil
}

func TestRouter(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Router Suite")
}

var _ = Describe("Router", func() {
	Context("When calling shouldReload", func() {
		Context("with an up-to-date mongo instance", func() {
			It("should return false", func() {
				rt := Router{}
				rt.mongoOpcounters = MongoOpcounters{0, 0, 0, 0, 0}

				mongoServerStatus := MongoServerStatus{}
				mongoServerStatus.Opcounters.Insert = 0
				mongoServerStatus.Opcounters.Update = 0
				mongoServerStatus.Opcounters.Delete = 0

				Expect(rt.shouldReload(mongoServerStatus)).To(
					Equal(false),
					"Router should determine no reload is necessary when Mongo opcounters haven't changed",
				)
			})
		})

		Context("with a stale mongo instance", func() {
			It("should return false when insert differs", func() {
				rt := Router{}
				rt.mongoOpcounters = MongoOpcounters{1, 1, 1, 1, 1}

				mongoServerStatus := MongoServerStatus{}
				mongoServerStatus.Opcounters.Insert = 2
				mongoServerStatus.Opcounters.Update = 1
				mongoServerStatus.Opcounters.Delete = 1

				Expect(rt.shouldReload(mongoServerStatus)).To(
					Equal(true),
					"Router should determine reload is necessary when Mongo insert counter has changed",
				)
			})

			It("should return false when update differs", func() {
				rt := Router{}
				rt.mongoOpcounters = MongoOpcounters{1, 1, 1, 1, 1}

				mongoServerStatus := MongoServerStatus{}
				mongoServerStatus.Opcounters.Insert = 1
				mongoServerStatus.Opcounters.Update = 2
				mongoServerStatus.Opcounters.Delete = 1

				Expect(rt.shouldReload(mongoServerStatus)).To(
					Equal(true),
					"Router should determine reload is necessary when Mongo update opcounter has changed",
				)
			})

		})
	})

	// Context("When calling getCurrentMongoInstance", func() {
	// 	It("should return error when unable to get the replica set", func() {
	// 		mockMongoObj := &mockMongoDB{
	// 			err: errors.New("Error connecting to replica set"),
	// 		}
	//
	// 		rt := Router{}
	// 		_, err := rt.getCurrentMongoInstance(mockMongoObj)
	//
	// 		Expect(err).NotTo(
	// 			BeNil(),
	// 			"Router should raise an error when it can't get replica set status from Mongo")
	// 	})
	//
	// 	It("should return fail to find an instance when the replica set status schema doesn't match the expected schema", func() {
	// 		replicaSetStatusBson := bson.M{"members": []bson.M{{"unknownProperty": "unknown"}}}
	// 		mockMongoObj := &mockMongoDB{
	// 			result: replicaSetStatusBson,
	// 		}
	//
	// 		rt := Router{}
	// 		_, err := rt.getCurrentMongoInstance(mockMongoObj)
	//
	// 		Expect(err).NotTo(
	// 			BeNil(),
	// 			"Router should raise an error when the current Mongo instance can't be found in the replica set status response")
	// 	})
	//
	// 	It("should return fail to find an instance when the replica set status contains no instances marked with self:true", func() {
	// 		replicaSetStatusBson := bson.M{"members": []bson.M{{"name": "mongo1", "self": false}}}
	// 		mockMongoObj := &mockMongoDB{
	// 			result: replicaSetStatusBson,
	// 		}
	//
	// 		rt := Router{}
	// 		_, err := rt.getCurrentMongoInstance(mockMongoObj)
	//
	// 		Expect(err).NotTo(
	// 			BeNil(),
	// 			"Router should raise an error when the current Mongo instance can't be found in the replica set status response")
	// 	})
	//
	// 	It("should return fail to find an instance when the replica set status contains multiple instances marked with self:true", func() {
	// 		replicaSetStatusBson := bson.M{"members": []bson.M{{"name": "mongo1", "self": true}, {"name": "mongo2", "self": true}}}
	// 		mockMongoObj := &mockMongoDB{
	// 			result: replicaSetStatusBson,
	// 		}
	//
	// 		rt := Router{}
	// 		_, err := rt.getCurrentMongoInstance(mockMongoObj)
	//
	// 		Expect(err).NotTo(
	// 			BeNil(),
	// 			"Router should raise an error when the replica set status response contains multiple current Mongo instances")
	// 	})
	//
	// 	It("should successfully return the current Mongo instance from the replica set", func() {
	// 		replicaSetStatusBson := bson.M{"members": []bson.M{{"name": "mongo1", "self": false}, {"name": "mongo2", "optime": 6945383634312364034, "self": true}}}
	// 		mockMongoObj := &mockMongoDB{
	// 			result: replicaSetStatusBson,
	// 		}
	//
	// 		expectedMongoInstance := MongoReplicaSetMember{
	// 			Name:    "mongo2",
	// 			Optime:  6945383634312364034,
	// 			Current: true,
	// 		}
	//
	// 		rt := Router{}
	// 		currentMongoInstance, _ := rt.getCurrentMongoInstance(mockMongoObj)
	//
	// 		Expect(currentMongoInstance).To(
	// 			Equal(expectedMongoInstance),
	// 			"Router should get the current Mongo instance from the replica set status response",
	// 		)
	// 	})
	// })
})
