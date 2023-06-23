package main

import (
	"testing"
	//"time"

	"database/sql"
	//"github.com/lib/pq"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestRouter(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Router Suite")
}

func sendNotifyToDatabase(db *sql.DB, channel string) error {
	_, err := db.Exec("NOTIFY " + channel)
	logInfo("notify?")
	return err
}

//var _ = Describe("Router", func() {
//	Context("When calling shouldReload", func() {
//		Context("with no update notification from postgres", func() {
//			It("should return false", func() {
//				rt := Router{}
//
//				listener := pq.NewListener("", time.Second, time.Minute, nil)
//				listener.Listen("events")
//
//				Expect(rt.shouldReload(listener)).To(
//					Equal(false),
//					"Router should determine no reload is necessary when it has not received a notification from the listener",
//				)
//			})
//		})
//
//		//Context("with a valid update notification from postgres", func() {
//		//	It("should return true", func() {
//		//
//		//	})
//		//})
//	})
//})
//
