package events

import (
	"context"
	"net/http"

	"github.com/go-openapi/runtime/middleware"
	"github.com/openshift/assisted-service/internal/common"
	eventsapi "github.com/openshift/assisted-service/internal/events/api"
	"github.com/openshift/assisted-service/models"
	logutil "github.com/openshift/assisted-service/pkg/log"
	"github.com/openshift/assisted-service/restapi"
	"github.com/openshift/assisted-service/restapi/operations/events"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

var _ restapi.EventsAPI = &Api{}

type Api struct {
	handler eventsapi.Handler
	log     logrus.FieldLogger
}

func NewApi(handler eventsapi.Handler, log logrus.FieldLogger) *Api {
	return &Api{
		handler: handler,
		log:     log,
	}
}

func (a *Api) V2ListEvents(ctx context.Context, params events.V2ListEventsParams) middleware.Responder {
	log := logutil.FromContext(ctx, a.log)
	V2getEventsParams := common.V2GetEventsParams{
		ClusterID:    params.ClusterID,
		HostIds:      params.HostIds,
		InfraEnvID:   params.InfraEnvID,
		Limit:        params.Limit,
		Offset:       params.Offset,
		Order:        params.Order,
		Severities:   params.Severities,
		Message:      params.Message,
		DeletedHosts: params.DeletedHosts,
		ClusterLevel: params.ClusterLevel,
		Categories:   params.Categories,
	}

	// DEPRECATED
	if params.HostID != nil {
		V2getEventsParams.HostIds = append(V2getEventsParams.HostIds, *params.HostID)
	}

	response, err := a.handler.V2GetEvents(ctx, &V2getEventsParams)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.NewApiError(http.StatusNotFound, err)
		}
		log.WithError(err).Errorf("failed to get events")
		return common.NewApiError(http.StatusInternalServerError, err)
	}

	evs := response.GetEvents()
	eventSeverityCount := response.GetEventSeverityCount()
	eventCount := response.GetEventCount()

	ret := make(models.EventList, len(evs))
	for i, ev := range evs {
		ret[i] = &models.Event{
			Name:       ev.Name,
			ClusterID:  ev.ClusterID,
			HostID:     ev.HostID,
			InfraEnvID: ev.InfraEnvID,
			Severity:   ev.Severity,
			EventTime:  ev.EventTime,
			Message:    ev.Message,
			Props:      ev.Props,
		}
	}

	return events.NewV2ListEventsOK().
		WithSeverityCountInfo((*eventSeverityCount)[models.EventSeverityInfo]).
		WithSeverityCountWarning((*eventSeverityCount)[models.EventSeverityWarning]).
		WithSeverityCountError((*eventSeverityCount)[models.EventSeverityError]).
		WithSeverityCountCritical((*eventSeverityCount)[models.EventSeverityCritical]).
		WithEventCount(*eventCount).
		WithPayload(ret)
}
