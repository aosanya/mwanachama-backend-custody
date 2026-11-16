package routes_test

import (
	"encoding/json"
	"net/http"
	"testing"
)

func getJSON(t *testing.T, url string) (int, map[string]any) {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return resp.StatusCode, out
}

func TestMineRouteReportsOnlyTheCallersOwnState(t *testing.T) {
	cm := steppingManager(t)
	publishMechanics(t, cm)
	resident := serve(t, cm, "resident-1")
	neighbour := serve(t, cm, "resident-2")

	if status, out := getJSON(t, resident.URL+"/consent/mine"); status != http.StatusOK || out["status"] != "none" {
		t.Fatalf("before agreeing = %d %v, want 200 none", status, out)
	}
	if status, out := postJSON(t, resident.URL+"/consent/agree", `{}`); status != http.StatusCreated {
		t.Fatalf("agree = %d %v", status, out)
	}
	status, out := getJSON(t, resident.URL+"/consent/mine")
	if status != http.StatusOK || out["status"] != "agreed" || out["agreed_at"] == "" || out["mechanics_version_id"] == "" {
		t.Fatalf("after agreeing = %d %v, want agreed with the time and the version", status, out)
	}
	if _, out := getJSON(t, neighbour.URL+"/consent/mine"); out["status"] != "none" {
		t.Fatalf("neighbour sees %v, want none", out)
	}
	if status, out := postJSON(t, resident.URL+"/consent/withdraw", `{}`); status != http.StatusCreated {
		t.Fatalf("withdraw = %d %v", status, out)
	}
	if _, out := getJSON(t, resident.URL+"/consent/mine"); out["status"] != "withdrawn" {
		t.Fatalf("after withdrawing = %v, want withdrawn", out)
	}
	if status, out := postJSON(t, resident.URL+"/consent/agree", `{}`); status != http.StatusCreated {
		t.Fatalf("agree again = %d %v", status, out)
	}
	if _, out := getJSON(t, resident.URL+"/consent/mine"); out["status"] != "agreed" {
		t.Fatalf("after agreeing again = %v, want agreed", out)
	}
}
