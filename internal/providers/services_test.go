package providers

import "testing"

func TestServiceValidation(t *testing.T) {
	for _, unit := range ServiceUnits() {
		for _, action := range []ServiceAction{ServiceStart, ServiceStop, ServiceRestart, ServiceEnable} {
			if err := (ServiceParams{Unit: unit, Action: action}).Validate(); err != nil {
				t.Fatalf("%s %s: %v", action, unit, err)
			}
		}
	}
	for _, params := range []ServiceParams{
		{},
		{Unit: "nginx", Action: ServiceStart},
		{Unit: "panel-agent.service", Action: ServiceStop},
		{Unit: "unlisted.service", Action: ServiceRestart},
		{Unit: "nginx.service", Action: "disable"},
		{Unit: "nginx.service", Action: "reload"},
		{Unit: "nginx.service", Action: ""},
	} {
		if err := params.Validate(); err == nil {
			t.Errorf("accepted %+v", params)
		}
	}
}

func TestServiceUnitsReturnsIndependentList(t *testing.T) {
	units := ServiceUnits()
	units[0] = "unlisted.service"
	if err := ServiceUnit("nginx.service").Validate(); err != nil {
		t.Fatal(err)
	}
	if err := units[0].Validate(); err == nil {
		t.Fatal("caller changed allowlist")
	}
}
