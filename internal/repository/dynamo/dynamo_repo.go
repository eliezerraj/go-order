package dynamo

import(
	"context"
	"time"
	"github.com/go-order/internal/erro"
	"github.com/go-order/internal/core"
	"github.com/go-order/internal/lib"
	"github.com/go-order/internal/config/config_aws"

	"github.com/rs/zerolog/log"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
)

var childLogger = log.With().Str("repository", "DynamoRepository").Logger()

type DynamoRepository struct {
	client 		*dynamodb.Client
	tableName   *string
}

func NewDynamoRepository(ctx context.Context, databaseDynamo core.DatabaseDynamo) (*DynamoRepository, error){
	childLogger.Debug().Msg("NewDynamoRepository")

	span := lib.Span(ctx, "repository.NewDynamoRepository")	
    defer span.End()

	sdkConfig, err :=config_aws.GetAWSConfig(ctx, databaseDynamo.AwsRegion)
	if err != nil{
		return nil, err
	}

	client := dynamodb.NewFromConfig(*sdkConfig)

	return &DynamoRepository {
		client: client,
		tableName: aws.String(databaseDynamo.OrderTableName),
	}, nil
}

func (r *DynamoRepository) Add(ctx context.Context, order core.Order) (*core.Order, error){
	childLogger.Debug().Msg("Add")

	span := lib.Span(ctx, "repo.Add")	
    defer span.End()

	order.PK 		= "ORDER-" + order.OrderID
	order.SK 		= "ORDER-" + order.OrderID
	order.CreateAt 	= time.Now()

	//log.Debug().Interface("======> order :",order).Msg("")
	/*item2 := map[string]types.AttributeValue{
        "pk": 		&types.AttributeValueMemberS{Value: "12345"},
        "sk":       	&types.AttributeValueMemberS{Value: "John Doe"},
        "Age":        &types.AttributeValueMemberN{Value: "30"},
        "IsActive":   &types.AttributeValueMemberBOOL{Value: true},
    }*/

	item, err := attributevalue.MarshalMap(order)
	if err != nil {
		childLogger.Error().Err(err).Msg("error MarshalMap")
		return nil, erro.ErrUnmarshal
	}

	//log.Debug().Interface(">>>>>>>>>>>>>>> item :",item).Msg("")

	putInput := &dynamodb.PutItemInput{
        TableName: r.tableName,
        Item:      item,
    }

	_, err = r.client.PutItem(ctx, putInput)
    if err != nil {
		childLogger.Error().Err(err).Msg("error PutItem")
		return nil, erro.ErrInsert
    }

	return &order, nil
}
