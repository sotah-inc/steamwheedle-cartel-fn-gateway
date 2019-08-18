package app

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"

	"cloud.google.com/go/compute/metadata"
	"github.com/sirupsen/logrus"
	"github.com/sotah-inc/steamwheedle-cartel/pkg/act"
	"github.com/sotah-inc/steamwheedle-cartel/pkg/blizzard"
	"github.com/sotah-inc/steamwheedle-cartel/pkg/logging"
	"github.com/sotah-inc/steamwheedle-cartel/pkg/logging/stackdriver"
	"github.com/sotah-inc/steamwheedle-cartel/pkg/sotah"
	"github.com/sotah-inc/steamwheedle-cartel/pkg/state/fn"
)

var port int
var serviceName string
var projectId string
var state fn.GatewayState

func init() {
	var err error

	// resolving project-id
	projectId, err = metadata.Get("project/project-id")
	if err != nil {
		log.Fatalf("Failed to get project-id: %s", err.Error())

		return
	}

	// resolving service name
	serviceName = os.Getenv("K_SERVICE")

	// establishing log verbosity
	logVerbosity, err := logrus.ParseLevel("info")
	if err != nil {
		logging.WithField("error", err.Error()).Fatal("Could not parse log level")

		return
	}
	logging.SetLevel(logVerbosity)

	// adding stackdriver hook
	logging.WithField("project-id", projectId).Info("Creating stackdriver hook")
	stackdriverHook, err := stackdriver.NewHook(projectId, serviceName)
	if err != nil {
		logging.WithFields(logrus.Fields{
			"error":     err.Error(),
			"projectID": projectId,
		}).Fatal("Could not create new stackdriver logrus hook")

		return
	}
	logging.AddHook(stackdriverHook)

	// done preliminary setup
	logging.WithField("service", serviceName).Info("Initializing service")

	// parsing http port
	port, err = strconv.Atoi(os.Getenv("PORT"))
	if err != nil {
		log.Fatalf("Failed to get port: %s", err.Error())

		return
	}
	logging.WithField("port", port).Info("Initializing with port")

	// producing gateway state
	logging.WithFields(logrus.Fields{
		"project":      projectId,
		"service-name": serviceName,
		"port":         port,
	}).Info("Producing fn-gateway state")

	state, err = fn.NewGatewayState(
		fn.GatewayStateConfig{ProjectId: projectId},
	)
	if err != nil {
		log.Fatalf("Failed to generate compute-live-auctions state: %s", err.Error())

		return
	}

	// fin
	logging.Info("Finished init")
}

func FnGateway(w http.ResponseWriter, r *http.Request) {
	logging.Info("Received request")

	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)

		return
	}

	switch r.URL.Path {
	case "/download-all-auctions":
		if err := state.DownloadAllAuctions(); err != nil {
			act.WriteErroneousErrorResponse(w, "Could not call download-all-auctions", err)

			logging.WithFields(logrus.Fields{
				"error": err.Error(),
			}).Error("Could not call download-all-auctions")

			return
		}

		w.WriteHeader(http.StatusCreated)
	case "/cleanup-all-manifests":
		if err := state.CleanupAllManifests(); err != nil {
			act.WriteErroneousErrorResponse(w, "Could not call cleanup-all-manifests", err)

			logging.WithFields(logrus.Fields{
				"error": err.Error(),
			}).Error("Could not call Could not call cleanup-all-manifests")

			return
		}

		w.WriteHeader(http.StatusOK)
	case "/cleanup-all-auctions":
		if err := state.CleanupAllAuctions(); err != nil {
			act.WriteErroneousErrorResponse(w, "Could not call cleanup-all-auctions", err)

			logging.WithFields(logrus.Fields{
				"error": err.Error(),
			}).Error("Could not call Could not call cleanup-all-auctions")

			return
		}

		w.WriteHeader(http.StatusOK)
	case "/compute-all-live-auctions":
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			act.WriteErroneousErrorResponse(w, "Could not read request body", err)

			logging.WithFields(logrus.Fields{
				"error": err.Error(),
			}).Error("Could not read request body")

			return
		}

		tuples, err := sotah.NewRegionRealmTimestampTuples(string(body))
		if err != nil {
			act.WriteErroneousErrorResponse(w, "Could not decode region-realm-timestamp tuples from request body", err)

			logging.WithFields(logrus.Fields{
				"error": err.Error(),
			}).Error("Could not decode region-realm-timestamp tuples from request body")

			return
		}

		if err := state.ComputeAllLiveAuctions(tuples); err != nil {
			act.WriteErroneousErrorResponse(w, "Could not call compute-all-live-auctions", err)

			logging.WithFields(logrus.Fields{
				"error": err.Error(),
			}).Error("Could not call compute-all-live-auctions")

			return
		}

		w.WriteHeader(http.StatusCreated)
	case "/compute-all-pricelist-histories":
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			act.WriteErroneousErrorResponse(w, "Could not read request body", err)

			logging.WithFields(logrus.Fields{
				"error": err.Error(),
			}).Error("Could not read request body")

			return
		}

		tuples, err := sotah.NewRegionRealmTimestampTuples(string(body))
		if err != nil {
			act.WriteErroneousErrorResponse(w, "Could not decode region-realm-timestamp tuples from request body", err)

			logging.WithFields(logrus.Fields{
				"error": err.Error(),
			}).Error("Could not decode region-realm-timestamp tuples from request body")

			return
		}

		if err := state.ComputeAllPricelistHistories(tuples); err != nil {
			act.WriteErroneousErrorResponse(w, "Could not call compute-all-pricelist-histories", err)

			logging.WithFields(logrus.Fields{
				"error": err.Error(),
			}).Error("Could not call compute-all-pricelist-histories")

			return
		}

		w.WriteHeader(http.StatusCreated)
	case "/sync-all-items":
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			act.WriteErroneousErrorResponse(w, "Could not read request body", err)

			logging.WithFields(logrus.Fields{
				"error": err.Error(),
			}).Error("Could not read request body")

			return
		}

		ids, err := blizzard.NewItemIds(string(body))
		if err != nil {
			act.WriteErroneousErrorResponse(w, "Could not decode item-ids from request body", err)

			logging.WithFields(logrus.Fields{
				"error": err.Error(),
			}).Error("Could not decode item-ids from request body")

			return
		}

		if err := state.SyncAllItems(ids); err != nil {
			act.WriteErroneousErrorResponse(w, "Could not call sync-all-items", err)

			logging.WithFields(logrus.Fields{
				"error": err.Error(),
			}).Error("Could not call sync-all-items")

			return
		}

		w.WriteHeader(http.StatusCreated)
	case "/cleanup-all-pricelist-histories":
		if err := state.CleanupAllPricelistHistories(); err != nil {
			act.WriteErroneousErrorResponse(w, "Could not call cleanup-all-pricelist-histories", err)

			logging.WithFields(logrus.Fields{
				"error": err.Error(),
			}).Error("Could not call Could not call cleanup-all-pricelist-histories")

			return
		}

		w.WriteHeader(http.StatusOK)
	}

	logging.Info("Sent response")
}
