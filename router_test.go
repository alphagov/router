package main

import (
	"errors"
	"testing"
	"time"

	"github.com/globalsign/mgo/bson"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type mockMongoDB struct {
	result bson.M
	err    error
}

func (m *mockMongoDB) Run(_ interface{}, res interface{}) error {
	if m.err != nil {
		return m.err
	}

	bytes, err := bson.Marshal(m.result)
	if err != nil {
		return err
	}

	err = bson.Unmarshal(bytes, res)
	if err != nil {
		return err
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
				initialOptime, _ := bson.NewMongoTimestamp(time.Date(2021, time.March, 12, 8, 0, 0, 0, time.UTC), 1)
				rt.mongoReadToOptime = initialOptime

				currentOptime, _ := bson.NewMongoTimestamp(time.Date(2021, time.March, 12, 8, 0, 0, 0, time.UTC), 1)
				mongoInstance := MongoReplicaSetMember{}
				mongoInstance.Optime = currentOptime

				Expect(rt.shouldReload(mongoInstance)).To(
					BeFalse(),
					"Router should determine no reload is necessary when Mongo optime hasn't changed",
				)
			})
		})

		Context("with a stale mongo instance", func() {
			It("should return false when timestamp differs", func() {
				rt := Router{}
				initialOptime, _ := bson.NewMongoTimestamp(time.Date(2021, time.March, 12, 8, 0, 0, 0, time.UTC), 1)
				rt.mongoReadToOptime = initialOptime

				currentOptime, _ := bson.NewMongoTimestamp(time.Date(2021, time.March, 12, 8, 2, 30, 0, time.UTC), 1)
				mongoInstance := MongoReplicaSetMember{}
				mongoInstance.Optime = currentOptime

				Expect(rt.shouldReload(mongoInstance)).To(
					BeTrue(),
					"Router should determine reload is necessary when Mongo optime has changed by timestamp",
				)
			})

			It("should return false when operand differs", func() {
				rt := Router{}
				initialOptime, _ := bson.NewMongoTimestamp(time.Date(2021, time.March, 12, 8, 0, 0, 0, time.UTC), 1)
				rt.mongoReadToOptime = initialOptime

				currentOptime, _ := bson.NewMongoTimestamp(time.Date(2021, time.March, 12, 8, 0, 0, 0, time.UTC), 2)
				mongoInstance := MongoReplicaSetMember{}
				mongoInstance.Optime = currentOptime

				Expect(rt.shouldReload(mongoInstance)).To(
					BeTrue(),
					"Router should determine reload is necessary when Mongo optime has changed by operand",
				)
			})
		})
	})

	Context("When calling getCurrentMongoInstance", func() {
		It("should return error when unable to get the replica set", func() {
			mockMongoObj := &mockMongoDB{
				err: errors.New("Error connecting to replica set"),
			}

			rt := Router{}
			_, err := rt.getCurrentMongoInstance(mockMongoObj)

			Expect(err).To(
				HaveOccurred(),
				"Router should raise an error when it can't get replica set status from Mongo")
		})

		It("should return fail to find an instance when the replica set status schema doesn't match the expected schema", func() {
			replicaSetStatusBson := bson.M{"members": []bson.M{{"unknownProperty": "unknown"}}}
			mockMongoObj := &mockMongoDB{
				result: replicaSetStatusBson,
			}

			rt := Router{}
			_, err := rt.getCurrentMongoInstance(mockMongoObj)

			Expect(err).To(
				HaveOccurred(),
				"Router should raise an error when the current Mongo instance can't be found in the replica set status response")
		})

		It("should return fail to find an instance when the replica set status contains no instances marked with self:true", func() {
			replicaSetStatusBson := bson.M{"members": []bson.M{{"name": "mongo1", "self": false}}}
			mockMongoObj := &mockMongoDB{
				result: replicaSetStatusBson,
			}

			rt := Router{}
			_, err := rt.getCurrentMongoInstance(mockMongoObj)

			Expect(err).To(
				HaveOccurred(),
				"Router should raise an error when the current Mongo instance can't be found in the replica set status response")
		})

		It("should return fail to find an instance when the replica set status contains multiple instances marked with self:true", func() {
			replicaSetStatusBson := bson.M{"members": []bson.M{{"name": "mongo1", "self": true}, {"name": "mongo2", "self": true}}}
			mockMongoObj := &mockMongoDB{
				result: replicaSetStatusBson,
			}

			rt := Router{}
			_, err := rt.getCurrentMongoInstance(mockMongoObj)

			Expect(err).To(
				HaveOccurred(),
				"Router should raise an error when the replica set status response contains multiple current Mongo instances")
		})

		It("should successfully return the current Mongo instance from the replica set", func() {
			replicaSetStatusBson := bson.M{"members": []bson.M{{"name": "mongo1", "self": false}, {"name": "mongo2", "optime": 6945383634312364034, "self": true}}}
			mockMongoObj := &mockMongoDB{
				result: replicaSetStatusBson,
			}

			expectedMongoInstance := MongoReplicaSetMember{
				Name:    "mongo2",
				Optime:  6945383634312364034,
				Current: true,
			}

			rt := Router{}
			currentMongoInstance, _ := rt.getCurrentMongoInstance(mockMongoObj)

			Expect(currentMongoInstance).To(
				Equal(expectedMongoInstance),
				"Router should get the current Mongo instance from the replica set status response",
			)
		})
	})
})
