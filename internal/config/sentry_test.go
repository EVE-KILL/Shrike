package config

import "testing"

func TestSentryConfiguration(t *testing.T) {
	clearEnv(t)
	c, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	if c.SentryDSN != "" || c.SentryEnvironment != "development" || c.SentryTracesSampleRate != 0.1 || !c.SentryLogs || !c.SentryMetrics {
		t.Fatalf("unexpected Sentry defaults")
	}
	t.Setenv("SENTRY_DSN", "https://test@example.com/42")
	t.Setenv("SENTRY_ENVIRONMENT", "production")
	t.Setenv("SENTRY_TRACES_SAMPLE_RATE", "1")
	t.Setenv("SENTRY_LOG_SAMPLE_RATE", "0.2")
	t.Setenv("SENTRY_METRICS", "false")
	c, err = Load("")
	if err != nil {
		t.Fatal(err)
	}
	if c.SentryEnvironment != "production" || c.SentryTracesSampleRate != 1 || c.SentryLogSampleRate != 0.2 || c.SentryMetrics || c.SourceOf("SentryDSN") != SourceEnv {
		t.Fatal("Sentry settings were not honored")
	}
}

func TestSentryRejectsInvalidRates(t *testing.T) {
	for _, value := range []string{"NaN", "Inf", "-0.1", "1.1", "invalid"} {
		t.Run(value, func(t *testing.T) {
			clearEnv(t)
			t.Setenv("SENTRY_TRACES_SAMPLE_RATE", value)
			if _, err := Load(""); err == nil {
				t.Fatal("accepted invalid sampling rate")
			}
		})
	}
}
