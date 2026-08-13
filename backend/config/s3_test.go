package config

import "testing"

func TestS3Config_PublicObjectURL(t *testing.T) {
	t.Parallel()

	cfg := S3Config{
		Endpoint:      "oss-cn-hangzhou.aliyuncs.com",
		Bucket:        "my-bucket",
		UseSSL:        true,
		PublicBaseURL: "https://cdn.example.com",
	}
	got := cfg.PublicObjectURL("kb1/file.png")
	want := "https://cdn.example.com/kb1/file.png"
	if got != want {
		t.Fatalf("PublicObjectURL() = %q, want %q", got, want)
	}
}

func TestS3Config_ResolveStaticFilePath(t *testing.T) {
	t.Parallel()

	cfg := S3Config{
		Endpoint:      "oss-cn-hangzhou.aliyuncs.com",
		Bucket:        "static-file",
		UseSSL:        true,
		PublicBaseURL: "https://static.example.com",
	}

	tests := []struct {
		in   string
		want string
	}{
		{"/static-file/kb1/a.png", "https://static.example.com/kb1/a.png"},
		{"https://already.absolute/x", "https://already.absolute/x"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := cfg.ResolveStaticFilePath(tt.in); got != tt.want {
			t.Fatalf("ResolveStaticFilePath(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestS3Config_IsExternalObjectStorage(t *testing.T) {
	t.Parallel()

	if !((&S3Config{Endpoint: "oss-cn-hangzhou.aliyuncs.com"}).IsExternalObjectStorage()) {
		t.Fatal("expected OSS endpoint to be external")
	}
	if (&S3Config{Endpoint: "panda-wiki-minio:9000"}).IsExternalObjectStorage() {
		t.Fatal("expected local minio endpoint not to be external")
	}
}
