package core

type DatabaseRDS struct {
    Host 				string `json:"host"`
    Port  				string `json:"port"`
	Schema				string `json:"schema"`
	DatabaseName		string `json:"databaseName"`
	User				string `json:"user"`
	Password			string `json:"password"`
	Db_timeout			int	`json:"db_timeout"`
	Postgres_Driver		string `json:"postgres_driver"`
}

type DatabaseDynamo struct {
	OrderTableName		string `json:"order_table"`
	AwsRegion			string	`json:"aws_region"`
}

type AppServer struct {
	InfoPod 		*InfoPod 		`json:"info_pod"`
	Server     		*Server     	`json:"server"`
	Database		*DatabaseRDS	`json:"database"`
	ConfigOTEL		*ConfigOTEL		`json:"otel_config"`
	QueueConfig		*QueueConfig	`json:"queue_config"`
	DynamoConfig	*DatabaseDynamo	`json:"dynamo_config"`
	BucketConfig	*BucketConfig	`json:"bucket_config"`
}

type InfoPod struct {
	PodName				string `json:"pod_name"`
	ApiVersion			string `json:"version"`
	OSPID				string `json:"os_pid"`
	IPAddress			string `json:"ip_address"`
	AvailabilityZone 	string `json:"availabilityZone"`
	IsAZ				bool	`json:"is_az"`
	AccountID			string `json:"account_id,omitempty"`
	Env					string `json:"enviroment,omitempty"`
	QueueType			string `json:"queue_type,omitempty"`
}

type Server struct {
	Port 			int `json:"port"`
	ReadTimeout		int `json:"readTimeout"`
	WriteTimeout	int `json:"writeTimeout"`
	IdleTimeout		int `json:"idleTimeout"`
	CtxTimeout		int `json:"ctxTimeout"`
}

type ConfigOTEL struct {
	OtelExportEndpoint		string
	TimeInterval            int64    `mapstructure:"TimeInterval"`
	TimeAliveIncrementer    int64    `mapstructure:"RandomTimeAliveIncrementer"`
	TotalHeapSizeUpperBound int64    `mapstructure:"RandomTotalHeapSizeUpperBound"`
	ThreadsActiveUpperBound int64    `mapstructure:"RandomThreadsActiveUpperBound"`
	CpuUsageUpperBound      int64    `mapstructure:"RandomCpuUsageUpperBound"`
	SampleAppPorts          []string `mapstructure:"SampleAppPorts"`
}

type QueueConfig struct {
	QueueUrl	string	`json:"queue_url"`
	AwsRegion	string	`json:"aws_region"`
}

type BucketConfig struct {
	BucketNameKey	string
	FilePath		string
	AwsRegion		string
}