package view

import (
    "context"
    "testing"
    "github.com/stretchr/testify/assert"

    domain "github.com/superchalupa/go-redfish/src/redfishresource"
    "github.com/superchalupa/go-redfish/src/ocp/model"
    "github.com/superchalupa/go-redfish/src/log15adapter"
)

func init() {
    log15adapter.InitializeApplicationLogging("")
}

func TestView(t *testing.T) {
    testModel := model.New( model.UpdateProperty("test", "HAPPY") )
    _  = New( 
        WithURI("TESTURI") ,
        WithModel("default", testModel) ,
        )

    var tests = []struct {
        testname string
        input domain.RedfishResourceProperty
        expected interface{}
    }{
        { "test string 1", domain.RedfishResourceProperty{Value: "appy", Meta: map[string]interface{}{
                "GET": map[string]interface{}{"plugin": "TESTURI", "property": "test", "model": "default"}}}, "HAPPY" },
        { "test recursion 1", 
            domain.RedfishResourceProperty{Value: map[string]interface{}{"happy":  domain.RedfishResourceProperty{Value: "joy"}}}, 
            map[string]interface{}{"happy": "joy"} },
    }
    for _, subtest := range tests {
        t.Run(subtest.testname, func(t *testing.T) {
            output, _ := domain.ProcessGET(context.Background(), subtest.input)
            assert.EqualValues(t, subtest.expected, output)
        })
    }
}
