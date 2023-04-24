package config

import (
	"encoding/json"
	"testing"

	"github.com/xigxog/kubefox-cli/internal/log"
	"github.com/xigxog/kubefox/libs/core/validator"
)

func TestConfig_Validation(t *testing.T) {
	cfg := &Config{
		GitHub: GitHub{
			Org: GitHubOrg{
				Id:   1,
				Name: "org",
				URL:  "https://github.com",
			},
			User: GitHubUser{
				Id:        1,
				Name:      "user",
				AvatarURL: "https://github.com",
				URL:       "https://github.com",
			},
			Token: "token",
		},
		KubeFox: KubeFox{
			URL:      "https://github.com",
			Platform: "platform",
		},
	}

	v := validator.New(log.Logger())
	if errs := v.Validate(cfg); errs != nil {
		j, _ := json.MarshalIndent(errs, "  ", "")
		t.Logf("config is invalid:\n%s", j)
		t.Fail()
	}
}
