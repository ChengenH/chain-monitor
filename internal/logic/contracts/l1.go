package contracts

import (
	"fmt"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"

	"github.com/scroll-tech/chain-monitor/internal/config"
	"github.com/scroll-tech/chain-monitor/internal/logic/contracts/abi/il1erc1155gateway"
	"github.com/scroll-tech/chain-monitor/internal/logic/contracts/abi/il1erc20gateway"
	"github.com/scroll-tech/chain-monitor/internal/logic/contracts/abi/il1erc721gateway"
	"github.com/scroll-tech/chain-monitor/internal/logic/contracts/abi/il1scrollmessenger"
	"github.com/scroll-tech/chain-monitor/internal/types"
)

type erc20GatewayMapping struct {
	tokenType types.ERC20
	address   common.Address
}

type l1Contracts struct {
	client *ethclient.Client

	messenger *il1scrollmessenger.Il1scrollmessenger

	erc20Gateways      map[types.ERC20]*il1erc20gateway.Il1erc20gateway
	erc20GatewayTokens []erc20GatewayMapping

	erc721Gateway         *il1erc721gateway.Il1erc721gateway
	erc721GatewayAddress  common.Address
	ERC1155Gateway        *il1erc1155gateway.Il1erc1155gateway
	ERC1155GatewayAddress common.Address
}

func newL1Contracts(c *ethclient.Client) *l1Contracts {
	return &l1Contracts{
		client:        c,
		erc20Gateways: make(map[types.ERC20]*il1erc20gateway.Il1erc20gateway),
	}
}

func (l *l1Contracts) register(conf *config.Config) error {
	var err error
	l.messenger, err = il1scrollmessenger.NewIl1scrollmessenger(conf.L1Config.L1Contracts.ScrollMessenger, l.client)
	if err != nil {
		log.Error("registerERC20Gateway failed", "address", conf.L1Config.L1Contracts.ScrollMessenger, "err", err)
		return fmt.Errorf("register l2 scroll messenger contract failed, address:%v, err:%w", conf.L1Config.L1Contracts.ScrollMessenger.Hex(), err)
	}

	erc20Gateways := []struct {
		address common.Address
		token   types.ERC20
	}{
		{conf.L1Config.L1Contracts.WETHGateway, types.WETH},
		{conf.L1Config.L1Contracts.StandardERC20Gateway, types.StandardERC20},
		{conf.L1Config.L1Contracts.CustomERC20Gateway, types.CustomERC20},
		{conf.L1Config.L1Contracts.DAIGateway, types.DAI},
		{conf.L1Config.L1Contracts.USDCGateway, types.USDC},
		{conf.L1Config.L1Contracts.LIDOGateway, types.LIDO},
		{conf.L1Config.L1Contracts.PufferGateway, types.PUFETH},
	}

	for _, gw := range erc20Gateways {
		if err := l.registerERC20Gateway(gw.address, gw.token); err != nil {
			log.Error("registerERC20Gateway failed", "address", gw.address, "token", gw.token, "err", err)
			return err
		}
	}

	erc721GatewayAddress := conf.L1Config.L1Contracts.ERC721Gateway
	if err := l.registerERC721Gateway(erc721GatewayAddress); err != nil {
		log.Error("registerERC721Gateway failed", "address", erc721GatewayAddress, "err", err)
		return err
	}

	erc1155GatewayAddress := conf.L1Config.L1Contracts.ERC1155Gateway
	if err := l.registerERC1155Gateway(erc1155GatewayAddress); err != nil {
		log.Error("registerERC1155Gateway failed", "address", erc1155GatewayAddress, "err", err)
		return err
	}

	return nil
}

func (l *l1Contracts) registerERC20Gateway(gatewayAddress common.Address, tokenType types.ERC20) error {
	if gatewayAddress == (common.Address{}) {
		log.Warn("l1 erc20 gateway unconfigured", "address", gatewayAddress, "token type", tokenType.String())
		return nil
	}
	erc20Gateway, err := il1erc20gateway.NewIl1erc20gateway(gatewayAddress, l.client)
	if err != nil {
		return fmt.Errorf("l1 register erc20 gateway contract failed, err:%w", err)
	}

	l.erc20Gateways[tokenType] = erc20Gateway
	l.erc20GatewayTokens = append(l.erc20GatewayTokens, erc20GatewayMapping{tokenType: tokenType, address: gatewayAddress})

	return nil
}

func (l *l1Contracts) registerERC721Gateway(gatewayAddress common.Address) error {
	if gatewayAddress == (common.Address{}) {
		log.Warn("l1 erc721 gateway unconfigured", "address", gatewayAddress)
		return nil
	}

	l.erc721GatewayAddress = gatewayAddress

	erc721Gateways, err := il1erc721gateway.NewIl1erc721gateway(gatewayAddress, l.client)
	if err != nil {
		return fmt.Errorf("l1 register erc721 gateway contract failed, err:%w", err)
	}
	l.erc721Gateway = erc721Gateways
	return nil
}

func (l *l1Contracts) registerERC1155Gateway(gatewayAddress common.Address) error {
	if gatewayAddress == (common.Address{}) {
		log.Warn("l1 erc1155 gateway unconfigured", "address", gatewayAddress)
		return nil
	}

	l.ERC1155GatewayAddress = gatewayAddress

	erc1155Gateways, err := il1erc1155gateway.NewIl1erc1155gateway(gatewayAddress, l.client)
	if err != nil {
		return fmt.Errorf("l1 register erc1155 gateway contract failed, err:%w", err)
	}
	l.ERC1155Gateway = erc1155Gateways
	return nil
}
