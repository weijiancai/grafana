package provisioning

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/grafana/grafana/pkg/infra/log"
	"github.com/grafana/grafana/pkg/services/ngalert/models"
	"github.com/grafana/grafana/pkg/services/ngalert/store"
	"github.com/grafana/grafana/pkg/services/sqlstore"
	"github.com/stretchr/testify/require"
)

func TestAlertRuleService(t *testing.T) {
	ruleService := createAlertRuleService(t)
	t.Run("alert rule creation should return the created id", func(t *testing.T) {
		var orgID int64 = 1
		rule, err := ruleService.CreateAlertRule(context.Background(), dummyRule("test#1", orgID), models.ProvenanceNone)
		require.NoError(t, err)
		require.NotEqual(t, 0, rule.ID, "expected to get the created id and not the zero value")
	})
	t.Run("alert rule creation should set the right provenance", func(t *testing.T) {
		var orgID int64 = 1
		rule, err := ruleService.CreateAlertRule(context.Background(), dummyRule("test#2", orgID), models.ProvenanceAPI)
		require.NoError(t, err)

		_, provenance, err := ruleService.GetAlertRule(context.Background(), orgID, rule.UID)
		require.NoError(t, err)
		require.Equal(t, models.ProvenanceAPI, provenance)
	})
	t.Run("alert rule group should be updated correctly", func(t *testing.T) {
		var orgID int64 = 1
		rule := dummyRule("test#3", orgID)
		rule.RuleGroup = "a"
		rule, err := ruleService.CreateAlertRule(context.Background(), rule, models.ProvenanceNone)
		require.NoError(t, err)
		require.Equal(t, int64(60), rule.IntervalSeconds)

		var interval int64 = 120
		err = ruleService.UpdateAlertGroup(context.Background(), orgID, rule.NamespaceUID, rule.RuleGroup, 120)
		require.NoError(t, err)

		rule, _, err = ruleService.GetAlertRule(context.Background(), orgID, rule.UID)
		require.NoError(t, err)
		require.Equal(t, interval, rule.IntervalSeconds)
	})
	t.Run("alert rule should get interval from existing rule group", func(t *testing.T) {
		var orgID int64 = 1
		rule := dummyRule("test#4", orgID)
		rule.RuleGroup = "b"
		rule, err := ruleService.CreateAlertRule(context.Background(), rule, models.ProvenanceNone)
		require.NoError(t, err)

		var interval int64 = 120
		err = ruleService.UpdateAlertGroup(context.Background(), orgID, rule.NamespaceUID, rule.RuleGroup, 120)
		require.NoError(t, err)

		rule = dummyRule("test#4-1", orgID)
		rule.RuleGroup = "b"
		rule, err = ruleService.CreateAlertRule(context.Background(), rule, models.ProvenanceNone)
		require.NoError(t, err)
		require.Equal(t, interval, rule.IntervalSeconds)
	})
	t.Run("alert rule provenace should be correctly checked", func(t *testing.T) {
		tests := []struct {
			name   string
			from   models.Provenance
			to     models.Provenance
			errNil bool
		}{
			{
				name:   "should be able to update from provenance none to api",
				from:   models.ProvenanceNone,
				to:     models.ProvenanceAPI,
				errNil: true,
			},
			{
				name:   "should be able to update from provenance none to file",
				from:   models.ProvenanceNone,
				to:     models.ProvenanceFile,
				errNil: true,
			},
			{
				name:   "should not be able to update from provenance api to file",
				from:   models.ProvenanceAPI,
				to:     models.ProvenanceFile,
				errNil: false,
			},
			{
				name:   "should not be able to update from provenance api to none",
				from:   models.ProvenanceAPI,
				to:     models.ProvenanceNone,
				errNil: false,
			},
			{
				name:   "should not be able to update from provenance file to api",
				from:   models.ProvenanceFile,
				to:     models.ProvenanceAPI,
				errNil: false,
			},
			{
				name:   "should not be able to update from provenance file to none",
				from:   models.ProvenanceFile,
				to:     models.ProvenanceNone,
				errNil: false,
			},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				var orgID int64 = 1
				rule := dummyRule(t.Name(), orgID)
				rule, err := ruleService.CreateAlertRule(context.Background(), rule, test.from)
				require.NoError(t, err)

				_, err = ruleService.UpdateAlertRule(context.Background(), rule, test.to)
				if test.errNil {
					require.NoError(t, err)
				} else {
					require.Error(t, err)
				}
			})
		}
	})
}

func createAlertRuleService(t *testing.T) AlertRuleService {
	t.Helper()
	sqlStore := sqlstore.InitTestDB(t)
	store := store.DBstore{
		SQLStore:     sqlStore,
		BaseInterval: time.Second * 10,
	}
	return AlertRuleService{
		ruleStore:       store,
		provenanceStore: store,
		xact:            sqlStore,
		log:             log.New("testing"),
		defaultInterval: 60,
	}
}

func dummyRule(title string, orgID int64) models.AlertRule {
	return models.AlertRule{
		OrgID:           orgID,
		Title:           title,
		Condition:       "A",
		Version:         1,
		IntervalSeconds: 60,
		Data: []models.AlertQuery{
			{
				RefID: "A",
				Model: json.RawMessage("{}"),
				RelativeTimeRange: models.RelativeTimeRange{
					From: models.Duration(60),
					To:   models.Duration(0),
				},
			},
		},
		RuleGroup:    "my-cool-group",
		For:          time.Second * 60,
		NoDataState:  models.OK,
		ExecErrState: models.OkErrState,
	}
}
