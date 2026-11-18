package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/viper"
)

type Config struct {
	Log           LogConfig    `mapstructure:"log"`
	HTTP          HTTPConfig   `mapstructure:"http"`
	Admin         AdminConfig  `mapstructure:"admin"`
	AdminPassword string       `mapstructure:"admin_password"`
	PG            PGConfig     `mapstructure:"pg"`
	RAG           RAGConfig    `mapstructure:"rag"`
	Redis         RedisConfig  `mapstructure:"redis"`
	Auth          AuthConfig   `mapstructure:"auth"`
	S3            S3Config     `mapstructure:"s3"`
	Sentry        SentryConfig `mapstructure:"sentry"`
	CaddyAPI      string       `mapstructure:"caddy_api"`
	SubnetPrefix  string       `mapstructure:"subnet_prefix"`
	RunWorker     bool         `mapstructure:"run_worker"`
}

type LogConfig struct {
	Level int `mapstructure:"level"`
}

type HTTPConfig struct {
	Port       int  `mapstructure:"port"`
	TrustProxy bool `mapstructure:"trust_proxy"`
}

type AdminConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Port    int    `mapstructure:"port"`
	DistDir string `mapstructure:"dist_dir"`
}

type PGConfig struct {
	DSN             string `mapstructure:"dsn"`
	MaxOpenConns    int    `mapstructure:"max_open_conns"`
	MaxIdleConns    int    `mapstructure:"max_idle_conns"`
	ConnMaxLifetime int    `mapstructure:"conn_max_lifetime"` // 秒
}

type RAGConfig struct {
	Provider string      `mapstructure:"provider"`
	PG       PGRAGConfig `mapstructure:"pg"`
}

type PGRAGConfig struct {
	EmbeddingDim    int `mapstructure:"embedding_dim"`
	ChunkTokenLimit int `mapstructure:"chunk_token_limit"`
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
}

type AuthConfig struct {
	Type string    `mapstructure:"type"`
	JWT  JWTConfig `mapstructure:"jwt"`
}

type JWTConfig struct {
	Secret string `mapstructure:"secret"`
}

type S3Config struct {
	Endpoint      string `mapstructure:"endpoint"`
	AccessKey     string `mapstructure:"access_key"`
	SecretKey     string `mapstructure:"secret_key"`
	Bucket        string `mapstructure:"bucket"`
	UseSSL        bool   `mapstructure:"use_ssl"`
	Region        string `mapstructure:"region"`
	PublicBaseURL string `mapstructure:"public_base_url"`
}

type SentryConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	DSN     string `mapstructure:"dsn"`
}

func NewConfig() (*Config, error) {
	defaultConfig := &Config{
		Log: LogConfig{
			Level: 0,
		},
		AdminPassword: "",
		HTTP: HTTPConfig{
			Port: 8000,
		},
		Admin: AdminConfig{
			Enabled: true,
			Port:    2443,
			DistDir: "/app/admin/dist",
		},
		PG: PGConfig{
			DSN:             "host=panda-wiki-postgres user=panda-wiki password=panda-wiki-secret dbname=panda-wiki port=5432 sslmode=disable TimeZone=Asia/Shanghai",
			MaxOpenConns:    50,
			MaxIdleConns:    10,
			ConnMaxLifetime: 3600,
		},
		RAG: RAGConfig{
			Provider: "pg",
			PG: PGRAGConfig{
				EmbeddingDim:    1024,
				ChunkTokenLimit: 512,
			},
		},
		Redis: RedisConfig{
			Addr:     "panda-wiki-redis:6379",
			Password: "",
		},
		Auth: AuthConfig{
			Type: "jwt",
			JWT:  JWTConfig{Secret: ""},
		},
		S3: S3Config{
			Endpoint:  "oss-cn-hangzhou.aliyuncs.com",
			AccessKey: "",
			SecretKey: "",
			Bucket:    "static-file",
			UseSSL:    true,
			Region:    "oss-cn-hangzhou",
		},
		Sentry: SentryConfig{
			Enabled: true,
			DSN:     "https://2a4cff1ae04b624ffc72663f523024ff@sentry.baizhi.cloud/4",
		},
		CaddyAPI:     "/app/run/caddy-admin.sock",
		SubnetPrefix: "169.254.15",
		RunWorker:    true,
	}

	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.SetConfigName("config")
	viper.SetConfigType("yml")

	// try to read config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// if config file not found, return default config
			return nil, err
		}
	}

	// merge config file values to default config
	if err := viper.Unmarshal(defaultConfig); err != nil {
		return nil, err
	}

	// finally, override sensitive info with env variables
	overrideWithEnv(defaultConfig)

	if err := defaultConfig.validate(); err != nil {
		return nil, err
	}

	return defaultConfig, nil
}

// validate 启动即校验关键配置，避免带缺失凭据的服务"健康"启动后在运行时才暴露
func (c *Config) validate() error {
	if c.Auth.Type == "" {
		c.Auth.Type = "jwt"
	}
	// 空 JWT secret 会用于 HS256 签发/校验，等同于允许任何人伪造管理员 token
	if c.Auth.JWT.Secret == "" {
		return fmt.Errorf("JWT_SECRET is required: refusing to start with an empty JWT secret (it would allow forging admin tokens)")
	}
	// 非致命缺失给出醒目警告，帮助部署排障（不阻塞启动）
	if c.S3.AccessKey == "" || c.S3.SecretKey == "" {
		fmt.Fprintln(os.Stderr, "warning: S3_ACCESS_KEY/S3_SECRET_KEY is empty — file upload will fail until configured")
	}
	if c.AdminPassword == "" {
		fmt.Fprintln(os.Stderr, "warning: ADMIN_PASSWORD is empty — the admin console has no password set")
	}
	return nil
}

// overrideWithEnv override sensitive info with env variables
func overrideWithEnv(c *Config) {
	if env := os.Getenv("POSTGRES_PASSWORD"); env != "" {
		c.PG.DSN = fmt.Sprintf("host=panda-wiki-postgres user=panda-wiki password=%s dbname=panda-wiki port=5432 sslmode=disable TimeZone=Asia/Shanghai", env)
	}
	if env := os.Getenv("REDIS_PASSWORD"); env != "" {
		c.Redis.Password = env
	}
	if env := os.Getenv("JWT_SECRET"); env != "" {
		c.Auth.JWT.Secret = env
	}
	if env := os.Getenv("S3_SECRET_KEY"); env != "" {
		c.S3.SecretKey = env
	}
	if env := os.Getenv("S3_ACCESS_KEY"); env != "" {
		c.S3.AccessKey = env
	}
	if env := os.Getenv("ADMIN_PASSWORD"); env != "" {
		c.AdminPassword = env
	}
	if env := os.Getenv("SUBNET_PREFIX"); env != "" {
		c.SubnetPrefix = env
	}
	// pg
	if env := os.Getenv("PG_DSN"); env != "" {
		c.PG.DSN = env
	}
	if env := os.Getenv("PG_MAX_OPEN_CONNS"); env != "" {
		if i, err := strconv.Atoi(env); err == nil && i > 0 {
			c.PG.MaxOpenConns = i
		}
	}
	if env := os.Getenv("PG_MAX_IDLE_CONNS"); env != "" {
		if i, err := strconv.Atoi(env); err == nil && i > 0 {
			c.PG.MaxIdleConns = i
		}
	}
	if env := os.Getenv("PG_CONN_MAX_LIFETIME"); env != "" {
		if i, err := strconv.Atoi(env); err == nil && i > 0 {
			c.PG.ConnMaxLifetime = i
		}
	}
	// rag — pgvector only
	c.RAG.Provider = "pg"
	if env := os.Getenv("RAG_PG_EMBEDDING_DIM"); env != "" {
		if i, err := strconv.Atoi(env); err == nil {
			c.RAG.PG.EmbeddingDim = i
		}
	}
	if env := os.Getenv("RAG_PG_CHUNK_TOKEN_LIMIT"); env != "" {
		if i, err := strconv.Atoi(env); err == nil {
			c.RAG.PG.ChunkTokenLimit = i
		}
	}
	// redis
	if env := os.Getenv("REDIS_ADDR"); env != "" {
		c.Redis.Addr = env
	}
	// s3
	if env := os.Getenv("S3_ENDPOINT"); env != "" {
		c.S3.Endpoint = env
	}
	if env := os.Getenv("S3_BUCKET"); env != "" {
		c.S3.Bucket = env
	}
	if env := os.Getenv("S3_USE_SSL"); env != "" {
		c.S3.UseSSL = env == "1" || env == "true" || env == "TRUE"
	}
	if env := os.Getenv("S3_REGION"); env != "" {
		c.S3.Region = env
	}
	if env := os.Getenv("S3_PUBLIC_BASE_URL"); env != "" {
		c.S3.PublicBaseURL = env
	}
	// sentry
	if env := os.Getenv("SENTRY_ENABLED"); env != "" {
		c.Sentry.Enabled = env == "1" || env == "true" || env == "TRUE"
	}
	if env := os.Getenv("SENTRY_DSN"); env != "" {
		c.Sentry.DSN = env
	}
	// caddy api
	if env := os.Getenv("CADDY_API"); env != "" {
		c.CaddyAPI = env
	}
	if env := os.Getenv("RUN_WORKER"); env != "" {
		c.RunWorker = env == "1" || env == "true" || env == "TRUE"
	}
	if env := os.Getenv("ADMIN_ENABLED"); env != "" {
		c.Admin.Enabled = env == "1" || env == "true" || env == "TRUE"
	}
	if env := os.Getenv("ADMIN_PORT"); env != "" {
		if i, err := strconv.Atoi(env); err == nil {
			c.Admin.Port = i
		}
	}
	if env := os.Getenv("ADMIN_DIST_DIR"); env != "" {
		c.Admin.DistDir = env
	}
	if env := os.Getenv("TRUST_PROXY"); env != "" {
		c.HTTP.TrustProxy = env == "1" || env == "true" || env == "TRUE"
	}
	// log level
	if env := os.Getenv("LOG_LEVEL"); env != "" {
		if i, err := strconv.Atoi(env); err == nil {
			// -4: debug
			// 0: info
			// 4: warn
			// 8: error
			c.Log.Level = i
		} else {
			fmt.Fprintf(os.Stderr, "Invalid log level: %s with err: %s\n", env, err)
		}
	}
}

func (*Config) GetString(key string) string {
	return viper.GetString(key)
}

func (*Config) GetInt(key string) int {
	return viper.GetInt(key)
}

func (*Config) GetUint64(key string) uint64 {
	return viper.GetUint64(key)
}

func (*Config) GetBool(key string) bool {
	return viper.GetBool(key)
}

func (*Config) GetStringSlice(key string) []string {
	return viper.GetStringSlice(key)
}

func (*Config) GetFloat64(key string) float64 {
	return viper.GetFloat64(key)
}
