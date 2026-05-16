package jobs

import "testing"

func TestPublicJobHidesInternalStorageFields(t *testing.T) {
	job := Job{
		ID:         "img_public",
		SpaceToken: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		Results: []Result{{
			Index:          0,
			OK:             true,
			ImageURL:       "/outputs/0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef/2026-05-17/img_public-01.png",
			OutputDate:     "2026-05-17",
			OutputFileName: "img_public-01.png",
		}},
	}

	public := PublicJob(job)
	if public.SpaceToken != "" {
		t.Fatalf("PublicJob leaked space token: %+v", public)
	}
	if public.Results[0].OutputDate != "" || public.Results[0].OutputFileName != "" {
		t.Fatalf("PublicJob leaked output metadata: %+v", public.Results[0])
	}
	if public.Results[0].ImageURL != "/api/background-tasks/img_public/images/0" {
		t.Fatalf("PublicJob did not rewrite image URL: %+v", public.Results[0])
	}
}
