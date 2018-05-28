package storages

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/allegro/akubra/httphandler"
	"github.com/allegro/akubra/log"
	"github.com/allegro/akubra/metrics"
	"github.com/allegro/akubra/types"
	"github.com/allegro/akubra/utils"
)

// SyncSender filters and writes inconsistencies to synclog
type SyncSender struct {
	AllowedMethods map[string]struct{}
	SyncLog        log.Logger
}

func (slf SyncSender) shouldResponseBeLogged(bresp BackendResponse) bool {
	if slf.AllowedMethods == nil {
		return false
	}
	_, allowed := slf.AllowedMethods[bresp.Response.Request.Method]
	if slf.SyncLog == nil || !allowed {
		return false
	}
	return true
}

func (slf SyncSender) send(success, failure BackendResponse) {
	if shouldBeFilteredInMaintenanceMode(success, failure) {
		return
	}

	errorMsg := emptyStrOrErrorMsg(failure.Error)
	contentLength := success.Response.ContentLength
	reqID := utils.RequestID(success.Response.Request)

	syncLogMsg := &httphandler.SyncLogMessageData{
		Method:        success.Response.Request.Method,
		FailedHost:    extractDestinationHostName(failure),
		SuccessHost:   extractDestinationHostName(success),
		Path:          success.Response.Request.URL.Path,
		AccessKey:     utils.ExtractAccessKey(success.Response.Request),
		UserAgent:     success.Response.Request.Header.Get("User-Agent"),
		ContentLength: contentLength,
		ErrorMsg:      errorMsg,
		ReqID:         reqID,
		Time:          time.Now().Format(time.RFC3339Nano),
	}

	metrics.Mark(fmt.Sprintf("reqs.inconsistencies.%s.method-%s", metrics.Clean(failure.Backend.Endpoint.Host), success.Response.Request.Method))
	logMsg, err := json.Marshal(syncLogMsg)
	if err != nil {
		log.Debugf("Marshall synclog error %s", err)
		return
	}
	slf.SyncLog.Println(string(logMsg))
}

func sendSynclogs(syncLog *SyncSender, success BackendResponse, failures []BackendResponse) {
	if len(failures) == 0 || (success == BackendResponse{}) || syncLog == nil || !syncLog.shouldResponseBeLogged(success) {
		return
	}
	for _, failure := range failures {
		syncLog.send(success, failure)
	}

}

func emptyStrOrErrorMsg(err error) string {
	if err != nil {
		return fmt.Sprintf("non nil error:%s", err)
	}
	return ""
}

func shouldBeFilteredInMaintenanceMode(success, failure BackendResponse) bool {
	if !failure.Backend.Maintenance {
		return false
	}
	isPutOrDelMethod := (success.Response.Request.Method == http.MethodPut) || (success.Response.Request.Method == http.MethodDelete)
	return !isPutOrDelMethod
}

// extractDestinationHostName extract destination hostname fromrequest
func extractDestinationHostName(r BackendResponse) string {
	if r.Response != nil {
		return r.Response.Request.URL.Host
	}
	berr, ok := r.Error.(*types.BackendError)
	if ok {
		return berr.Backend()
	}
	log.Printf("Requested backend is not retrievable from tuple %#v", r)
	return ""
}