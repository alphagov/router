package router

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SchemaMap", func() {
	Context("When calling schemaMap", func() {
		It("should make a schema to backend map", func() {
			schema_map := schemaMap()

			Expect(schema_map).To(HaveKey("answer"))
			Expect(schema_map["answer"]).To(Equal("frontend"))
		})
	})
})
