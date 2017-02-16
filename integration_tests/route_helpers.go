package integration

import (
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = AfterEach(func() {
	clearRoutes()
})

var (
	routerDB *mgo.Database
)

type Redirect struct {
	Path         string `bson:"path"`
	Type         string `bson:"type"`
	Destination  string `bson:"destination"`
	SegmentsMode string `bson:"segments_mode"`
	RedirectType string `bson:"redirect_type"`
	Disabled     bool   `bson:"disabled"`
}

type Route struct {
	Path     string `bson:"path"`
	Type     string `bson:"type"`
	Disabled bool   `bson:"disabled"`
}

type ContentItem struct {
	RenderingApp string     `bson:"rendering_app"`
	DocumentType string     `bson:"document_type"`
	Routes       []Route    `bson:routes`
	Redirects    []Redirect `bson:redirects`
}

func NewBackendRoute(backendID string, extraParams ...string) ContentItem {
	route := Route{}
	if len(extraParams) > 0 {
		route.Type = extraParams[0]
	}

	contentItem := ContentItem{
		RenderingApp: backendID,
		DocumentType: "publication",
		Routes:       []Route{route},
	}

	return contentItem
}

func NewRedirectRoute(redirectTo string, extraParams ...string) ContentItem {
	redirect := Redirect{
		Destination:  redirectTo,
		RedirectType: "permanent",
		Type:         "exact",
	}

	if len(extraParams) > 0 {
		redirect.Type = extraParams[0]
	}
	if len(extraParams) > 1 {
		redirect.RedirectType = extraParams[1]
	}
	if len(extraParams) > 2 {
		redirect.SegmentsMode = extraParams[2]
	}
	contentItem := ContentItem{
		DocumentType: "redirect",
		Redirects:    []Redirect{redirect},
	}

	return contentItem
}

func NewGoneRoute(extraParams ...string) ContentItem {
	route := Route{}
	if len(extraParams) > 0 {
		route.Type = extraParams[0]
	}

	contentItem := ContentItem{
		DocumentType: "gone",
		Routes:       []Route{route},
	}

	return contentItem
}

func init() {
	sess, err := mgo.Dial("localhost")
	if err != nil {
		panic("Failed to connect to mongo: " + err.Error())
	}
	routerDB = sess.DB("content_store_test")
}

func addBackend(id, url string) {
	err := routerDB.C("backends").Insert(bson.M{"backend_id": id, "backend_url": url})
	Expect(err).To(BeNil())
}

func addRoute(path string, contentItem ContentItem) {
	if contentItem.DocumentType == "redirect" {
		contentItem.Redirects[0].Path = path
	} else {
		contentItem.Routes[0].Path = path
	}

	err := routerDB.C("content_items").Insert(contentItem)
	Expect(err).To(BeNil())
}

func clearRoutes() {
	routerDB.C("content_items").DropCollection()
	routerDB.C("backends").DropCollection()
}
