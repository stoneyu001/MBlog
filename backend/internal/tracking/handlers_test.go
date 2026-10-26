package tracking

import "testing"

func TestValidateTrackRequestAcceptsPageLeave(t *testing.T) {
	req := UnpartitionedTrackEventRequest{
		EventType:     "page_leave",
		PagePath:      "/tech/example",
		EventDuration: 12,
	}
	if err := validateTrackRequest(&req); err != nil {
		t.Fatalf("合法事件被拒绝: %v", err)
	}
	if req.EventType != "PAGE_LEAVE" {
		t.Fatalf("事件类型未规范化: %q", req.EventType)
	}
}

func TestValidateTrackRequestRejectsInvalidData(t *testing.T) {
	tests := []UnpartitionedTrackEventRequest{
		{EventType: "UNKNOWN", PagePath: "/"},
		{EventType: "PAGEVIEW", PagePath: ""},
		{EventType: "PAGE_LEAVE", PagePath: "/", EventDuration: 3601},
	}
	for _, req := range tests {
		request := req
		if err := validateTrackRequest(&request); err == nil {
			t.Fatalf("非法事件未被拒绝: %+v", req)
		}
	}
}
