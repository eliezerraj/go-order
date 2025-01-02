package core

import (
	"time"
)

type DatabaseRDS struct {
    Host 				string `json:"host"`
    Port  				string `json:"port"`
	Schema				string `json:"schema"`
	DatabaseName		string `json:"databaseName"`
	User				string `json:"user"`
	Password			string `json:"password"`
	Db_timeout			int	`json:"db_timeout"`
	Postgres_Driver		string `json:"postgres_driver"`
	CertPem	    		string 
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

type Order struct {
	ID				int			`json:"id,omitempty"`
	PK				string		`dynamodbav:"pk"`
	SK				string		`dynamodbav:"sk"`
	OrderID			string  	`json:"order_id,omitempty"`
	PersonID		string  	`json:"person_id,omitempty"`
	ProductID		[]string  	`json:"products_id,omitempty"`
	Status			string  	`json:"status,omitempty"`
	Currency		string  	`json:"currency,omitempty"`
	Amount			float64 	`json:"amount,omitempty"`
	CreateAt		time.Time 	`json:"create_at,omitempty"`
	UpdateAt		*time.Time 	`json:"update_at,omitempty"`
	TenantID		string  	`json:"tenant_id,omitempty"`
}

type Event struct {
	Key			string      `json:"key"`
    EventDate   time.Time   `json:"event_date"`
    EventType   string      `json:"event_type"`
    EventData   *EventData   `json:"event_data"`
}

type EventData struct {
    Order   *Order    `json:"order"`
}

type OrderFile struct {
	Name	string	`json:"file_name,omitempty"`
	File	[]byte	`json:"file,omitempty"`
}