package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-chi/chi"
	"github.com/jinzhu/gorm"

	"github.com/CMSgov/bcda-app/ssas"
	"github.com/CMSgov/bcda-app/ssas/service"
	"github.com/CMSgov/bcda-app/ssas/service/admin"
	"github.com/CMSgov/bcda-app/ssas/service/public"
)

var doMigrate bool
var doAddFixtureData bool
var doResetSecret bool
var doNewAdminSystem bool
var doMigrateAndStart bool
var doStart bool
var clientID string
var systemName string
var output io.Writer

func init() {
	const usageMigrate = "unconditionally migrate the db"
	flag.BoolVar(&doMigrate, "migrate", false, usageMigrate)

	const usageAddFixtureData = "unconditionally add fixture data"
	flag.BoolVar(&doAddFixtureData, "add-fixture-data", false, usageAddFixtureData)

	const usageResetSecret = "reset system secret for the given client_id; requires client-id flag with argument"
	flag.BoolVar(&doResetSecret, "reset-secret", false, usageResetSecret)
	flag.StringVar(&clientID, "client-id", "", "a system's client id")

	const usageNewAdminSystem = "add a new admin system to the service; requires system-name flag with argument"
	flag.BoolVar(&doNewAdminSystem, "new-admin-system", false, usageNewAdminSystem)
	flag.StringVar(&systemName, "system-name", "", "the system's name (e.g., 'BCDA Admin')")

	// we need this all-in-one command to start using fresh in the docker container
	const usageMigrateAndStart = "start the service; if DEBUG=true, will also migrate the db"
	flag.BoolVar(&doMigrateAndStart, "migrate-and-start", false, usageMigrateAndStart)

	const usageStart = "start the service"
	flag.BoolVar(&doStart, "start", false, usageStart)
}

// We provide some simple commands for bootstrapping the system into place. Commands cannot be combined.
func main() {
	ssas.Logger.Info("Home of the System-to-System Authentication Service")
	output = os.Stdout

	flag.Parse()
	if doMigrate {
		ssas.InitializeGroupModels()
		ssas.InitializeSystemModels()
		ssas.InitializeBlacklistModels()
		return
	}
	if doAddFixtureData {
		addFixtureData()
		return
	}
	if doResetSecret && clientID != "" {
		resetSecret(clientID)
		return
	}
	if doNewAdminSystem && systemName != "" {
		newAdminSystem(systemName)
		return
	}
	if doMigrateAndStart {
		if os.Getenv("DEBUG") == "true" {
			ssas.InitializeGroupModels()
			ssas.InitializeSystemModels()
			ssas.InitializeBlacklistModels()
			addFixtureData()
		}
		start()
		return
	}
	if doStart {
		start()
		return
	}
}

func start() {
	ssas.Logger.Infof("%s", "Starting ssas...")

	ps := public.Server()
	if ps == nil {
		ssas.Logger.Error("unable to create public server")
		os.Exit(-1)
	}
	ps.LogRoutes()
	ps.Serve()

	as := admin.Server()
	if as == nil {
		ssas.Logger.Error("unable to create admin server")
		os.Exit(-1)
	}
	as.LogRoutes()
	as.Serve()

	service.StartBlacklist()

	// Accepts and redirects HTTP requests to HTTPS. Not sure we should do this.
	forwarder := &http.Server{
		Handler:      newForwardingRouter(),
		Addr:         ":3005",
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}
	ssas.Logger.Fatal(forwarder.ListenAndServe())
}

func newForwardingRouter() http.Handler {
	r := chi.NewRouter()
	r.Use(service.NewAPILogger(), service.ConnectionClose)
	r.Get("/*", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// TODO only forward requests for paths in our own host or resource server
		url := "https://" + req.Host + req.URL.String()
		ssas.Logger.Infof("forwarding from %s to %s", req.Host+req.URL.String(), url)
		http.Redirect(w, req, url, http.StatusMovedPermanently)
	}))
	return r
}

func addFixtureData() {
	db := ssas.GetGORMDbConnection()
	defer ssas.Close(db)

	if err := db.Save(&ssas.Group{GroupID: "admin"}).Error; err != nil {
		fmt.Println(err)
	}
	// group for cms_id A9994; client_id 0c527d2e-2e8a-4808-b11d-0fa06baf8254
	if err := db.Save(&ssas.Group{GroupID: "0c527d2e-2e8a-4808-b11d-0fa06baf8254", Data: ssas.GroupData{GroupID: "0c527d2e-2e8a-4808-b11d-0fa06baf8254"}, XData: `"{\"cms_ids\":[\"A9994\"]}"`}).Error; err != nil {
		fmt.Println(err)
	}
	makeSystem(db, "admin", "31e029ef-0e97-47f8-873c-0e8b7e7f99bf",
		"BCDA API Admin", "bcda-admin",
		"nbZ5oAnTlzyzeep46bL4qDGGuidXuYxs3xknVWBKjTI=:9s/Tnqvs8M7GN6VjGkLhCgjmS59r6TaVguos8dKV9lGqC1gVG8ywZVEpDMkdwOaj8GoNe4TU3jS+OZsK3kTfEQ==",
	)
	makeSystem(db, "0c527d2e-2e8a-4808-b11d-0fa06baf8254",
		"0c527d2e-2e8a-4808-b11d-0fa06baf8254", "ACO Dev", "bcda-api",
		"h5hF9cm0Wmm+ClnoF0+Dq5JCQFmDVtzAsaquigoYcTk=:mptcWsBLNYFylRT1q95brbfKiaQkUt8oZXml0EMXobghbVVewZeG40EfNqe10u1+RftftEMvzSCB/oG17MRpVA==")
}

func makeSystem(db *gorm.DB, groupID, clientID, clientName, scope, hash string) {
	pem := `-----BEGIN PUBLIC KEY-----
	MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEArhxobShmNifzW3xznB+L
	I8+hgaePpSGIFCtFz2IXGU6EMLdeufhADaGPLft9xjwdN1ts276iXQiaChKPA2CK
	/CBpuKcnU3LhU8JEi7u/db7J4lJlh6evjdKVKlMuhPcljnIKAiGcWln3zwYrFCeL
	cN0aTOt4xnQpm8OqHawJ18y0WhsWT+hf1DeBDWvdfRuAPlfuVtl3KkrNYn1yqCgQ
	lT6v/WyzptJhSR1jxdR7XLOhDGTZUzlHXh2bM7sav2n1+sLsuCkzTJqWZ8K7k7cI
	XK354CNpCdyRYUAUvr4rORIAUmcIFjaR3J4y/Dh2JIyDToOHg7vjpCtNnNoS+ON2
	HwIDAQAB
	-----END PUBLIC KEY-----`
	system := ssas.System{GroupID: groupID, ClientID: clientID, ClientName: clientName, APIScope: scope}
	if err := db.Save(&system).Error; err != nil {
		ssas.Logger.Warn(err)
	}

	encryptionKey := ssas.EncryptionKey{
		Body:     pem,
		SystemID: system.ID,
	}
	if err := db.Save(&encryptionKey).Error; err != nil {
		ssas.Logger.Warn(err)
	}

	secret := ssas.Secret{
		Hash:     hash,
		SystemID: system.ID,
	}
	if err := db.Save(&secret).Error; err != nil {
		ssas.Logger.Warn(err)
	}
}

func resetSecret(clientID string) {
	var (
		err error
		s   ssas.System
		c   ssas.Credentials
	)
	if s, err = ssas.GetSystemByClientID(clientID); err != nil {
		ssas.Logger.Warn(err)
	}
	ssas.OperationCalled(ssas.Event{Op: "ResetSecret", TrackingID: cliTrackingID(), Help: "calling from main.resetSecret()"})
	if c, err = s.ResetSecret(clientID); err != nil {
		ssas.Logger.Warn(err)
	} else {
		_, _ = fmt.Fprintf(output, "%s\n", c.ClientSecret)
	}
}

func newAdminSystem(name string) {
	var (
		err error
		pk  string
		c   ssas.Credentials
		u   uint64
	)
	if pk, err = ssas.GeneratePublicKey(2048); err != nil {
		ssas.Logger.Errorf("no public key; %s", err)
		return
	}

	trackingID := cliTrackingID()
	ssas.OperationCalled(ssas.Event{Op: "RegisterSystem", TrackingID: trackingID, Help: "calling from main.newAdminSystem()"})
	if c, err = ssas.RegisterSystem(name, "admin", "bcda-api", pk, trackingID); err != nil {
		ssas.Logger.Error(err)
		return
	}

	if u, err = strconv.ParseUint(c.SystemID, 10, 64); err != nil {
		ssas.Logger.Errorf("invalid systemID %d; %s", u, err)
		return
	}

	db := ssas.GetGORMDbConnection()
	defer db.Close()
	if err = db.Model(&ssas.System{}).Where("id = ?", uint(u)).Update("api_scope", "bcda-admin").Error; err != nil {
		ssas.Logger.Warnf("bcda-admin scope not set for new system %s", c.SystemID)
	} else {
		_, _ = fmt.Fprintf(output, "%s\n", c.ClientID)
	}
}

func cliTrackingID() string {
	return fmt.Sprintf("cli-command-%d", time.Now().Unix())
}
