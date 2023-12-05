package messagematch

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/scroll-tech/chain-monitor/internal/config"
	"github.com/scroll-tech/chain-monitor/internal/logic/slack"
	"github.com/scroll-tech/chain-monitor/internal/orm"
	"github.com/scroll-tech/chain-monitor/internal/types"
	"github.com/scroll-tech/chain-monitor/internal/utils"
)

// LogicMessageMatch defines the logic related to message matching.
type LogicMessageMatch struct {
	db                       *gorm.DB
	conf                     *config.Config
	gatewayMessageMatchOrm   *orm.GatewayMessageMatch
	messengerMessageMatchOrm *orm.MessengerMessageMatch
}

// NewMessageMatchLogic initializes a new instance of Logic with an instance of orm.GatewayMessageMatch/orm.MessengerMessageMatch
func NewMessageMatchLogic(cfg *config.Config, db *gorm.DB) *LogicMessageMatch {
	return &LogicMessageMatch{
		db:                       db,
		conf:                     cfg,
		gatewayMessageMatchOrm:   orm.NewGatewayMessageMatch(db),
		messengerMessageMatchOrm: orm.NewMessengerMessageMatch(db),
	}
}

// GetLatestBlockNumber retrieves the latest block number for a given layer type.
func (t *LogicMessageMatch) GetLatestBlockNumber(ctx context.Context, layer types.LayerType) (uint64, error) {
	blockValidMessageMatch, blockValidErr := t.messengerMessageMatchOrm.GetLatestBlockValidMessageMatch(ctx, layer)
	if blockValidErr != nil {
		return 0, blockValidErr
	}

	if layer == types.Layer1 && blockValidMessageMatch == nil {
		return t.conf.L1Config.StartNumber, nil
	}

	if layer == types.Layer2 && blockValidMessageMatch == nil {
		return 0, nil
	}

	var number uint64
	switch layer {
	case types.Layer1:
		number = blockValidMessageMatch.L1BlockNumber
	case types.Layer2:
		number = blockValidMessageMatch.L2BlockNumber
	}

	return number, nil
}

// InsertOrUpdateMessageMatches insert or update the gateway/messenger event info
func (t *LogicMessageMatch) InsertOrUpdateMessageMatches(ctx context.Context, layer types.LayerType, gatewayMessageMatches []orm.GatewayMessageMatch, messengerMessageMatches []orm.MessengerMessageMatch) error {
	var effectRows int64
	err := t.db.Transaction(func(tx *gorm.DB) error {
		for _, message := range messengerMessageMatches {
			if layer == types.Layer1 {
				message.L1BlockStatus = int(types.BlockStatusTypeValid)
				message.L1BlockStatusUpdatedAt = utils.NowUTC()
			} else {
				message.L2BlockStatus = int(types.BlockStatusTypeValid)
				message.L2BlockStatusUpdatedAt = utils.NowUTC()
			}
			effectRow, err := t.messengerMessageMatchOrm.InsertOrUpdateEventInfo(ctx, layer, message, tx)
			if err != nil {
				return fmt.Errorf("messenger event orm insert failed, err: %w, layer:%s", err, layer.String())
			}

			if effectRow == 0 {
				slack.Notify(slack.MrkDwnMessengerMessageMatchDuplicated(layer, message))
				return fmt.Errorf("messenger event orm insert duplicated")
			}
			effectRows += effectRow
		}

		for _, message := range gatewayMessageMatches {
			if layer == types.Layer1 {
				message.L1BlockStatus = int(types.BlockStatusTypeValid)
				message.L1BlockStatusUpdatedAt = utils.NowUTC()
			} else {
				message.L2BlockStatus = int(types.BlockStatusTypeValid)
				message.L2BlockStatusUpdatedAt = utils.NowUTC()
			}
			effectRow, err := t.gatewayMessageMatchOrm.InsertOrUpdateEventInfo(ctx, layer, message, tx)
			if err != nil {
				return fmt.Errorf("gateway event orm insert failed, err: %w, layer:%s", err, layer.String())
			}

			if effectRow == 0 {
				slack.Notify(slack.MrkDwnGatewayMessageMatchDuplicated(layer, message))
				return fmt.Errorf("gateway event orm insert duplicated")
			}
			effectRows += effectRow
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("insert or update event info failed, err:%w", err)
	}

	if int(effectRows) != len(messengerMessageMatches)+len(gatewayMessageMatches) {
		return fmt.Errorf("gateway and messenger event orm insert failed, effectRow:%d not equal messageMatches:%d", effectRows, len(messengerMessageMatches)+len(gatewayMessageMatches))
	}
	return nil
}
