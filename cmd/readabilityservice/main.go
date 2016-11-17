package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	restful "github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful/swagger"
	"github.com/the42/readability"
)

type ReadabiltySimpleRequest struct {
	CheckString     *string `description:"Input String whose readability should be checked"`
	CorrelationID   *string `description:"request provided CorrelationID copied to response for requests/response matchmaking"`
	ReadabilityType *string `description:"Algorithm to use for readability check"`
}
type ReadabilitySimpleResponse struct {
	ReadabiltySimpleRequest `description:"Copied over from Request without CheckString"`
	Response                struct {
		Readability float32 `description:"Readability score result"`
		Message     *string `description:"diagnostic message returned by readability ccheck"`
		StatusCode  int     `description:"0:success, -1: no success, check Message"`
	}
}

var readabilityrequesttypemappings = map[string]readability.CompareType{
	"WSTF1": readability.WSTF1,
	"WSTF2": readability.WSTF2,
	"WSTF3": readability.WSTF3,
	"WSTF4": readability.WSTF4,
}

type readabilityservice struct {
	r *readability.Readability
}

func (s *readabilityservice) readabilityservice(request *restful.Request, response *restful.Response) {

	readabilityrequest := ReadabiltySimpleRequest{}
	if err := request.ReadEntity(&readabilityrequest); err != nil {
		logresponse(response, http.StatusBadRequest, fmt.Sprintf("unable to parse request: %s", err.Error()))
		return
	}
	if readabilityrequest.CheckString == nil {
		logresponse(response, http.StatusBadRequest, fmt.Sprintf("ReadabilitySimpleRequest.CheckString is required but not set"))
		return
	}

	var readability_type readability.CompareType

	if readabilityrequest.ReadabilityType != nil && len(*readabilityrequest.ReadabilityType) > 0 {
		readability_type = readabilityrequesttypemappings[*readabilityrequest.ReadabilityType]
	} else {
		readability_type = readability.WSTF1
	}

	result := ReadabilitySimpleResponse{ReadabiltySimpleRequest: readabilityrequest}
	// set the input string to nil for performance reasons. May correlate result to request by using CorrelationID
	result.ReadabiltySimpleRequest.CheckString = nil

	switch readability_type {
	case readability.WSTF1, readability.WSTF2, readability.WSTF3, readability.WSTF4:
		readabilityresult, err := s.r.WienerSachTextFormelType(*readabilityrequest.CheckString, readability_type)
		if err != nil {
			logresponse(response, http.StatusBadRequest, fmt.Sprintf("WienerSachTextFormelType returned error: %s", err.Error()))
			return
		}
		result.Response.Readability = readabilityresult
	default:
		result.Response.StatusCode = -1
		s := "no method found to perform readability check"
		result.Response.Message = &s
	}
	response.WriteAsJson(result)
}

func logresponse(resp *restful.Response, code int, message string) {
	resp.WriteErrorString(code, message)
	log.Print(message)
}

func main() {
	ws := new(restful.WebService).
		Produces(restful.MIME_JSON).
		Consumes(restful.MIME_JSON)

	//BEGIN: CORS support
	/*
		cors := restful.CrossOriginResourceSharing{
			ExposeHeaders:  []string{"X-My-Header"},
			AllowedHeaders: []string{"Content-Type", "Accept"},
			AllowedMethods: []string{"GET", "POST", "PUT"},
			CookiesAllowed: false,
			Container:      restful.DefaultContainer}

		restful.DefaultContainer.Filter(cors.Filter)
		// Add container filter to respond to OPTIONS
		restful.DefaultContainer.Filter(restful.DefaultContainer.OPTIONSFilter)
	*/
	//END: CORS support

	s := &readabilityservice{}
	if r, err := readability.NewReadability("de"); err == nil {
		s.r = r
	} else {
		log.Fatalf("Cannot create NewReadability Instance: %s\n", err.Error())
		return
	}

	ws.Route(ws.PUT("/readability").
		To(s.readabilityservice).
		Produces(restful.MIME_JSON).
		Consumes(restful.MIME_JSON).
		Reads(ReadabiltySimpleRequest{}).
		Returns(http.StatusOK, "", ReadabilitySimpleResponse{}).
		Returns(http.StatusInternalServerError, "", nil).
		Returns(http.StatusBadRequest, "", nil))
	restful.Add(ws)

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	hostname := os.Getenv("HOSTNAME")

	config := swagger.Config{
		WebServices:     restful.DefaultContainer.RegisteredWebServices(),
		ApiPath:         "/apidocs/apidocs.json",
		SwaggerPath:     "/swagger/",
		SwaggerFilePath: "./swagger-ui/dist"}
	swagger.RegisterSwaggerService(config, restful.DefaultContainer)

	log.Fatal(http.ListenAndServe(hostname+":"+port, nil))
}