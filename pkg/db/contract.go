package db

import (
	"database/sql"
	"time"

	uuid "github.com/iris-contrib/go.uuid"
)

type Contract struct {
	Id                    uint           `gorm:"primaryKey" json:"id"`
	ProjectId             uuid.UUID      `json:"projectId"`
	WorkflowId            uint           `json:"workflowId"`
	WorkflowDetailId      uint           `json:"workflowDetailId"`
	Name                  string         `json:"name"`
	Version               string         `json:"version"`
	Network               sql.NullString `json:"network"`
	BuildTime             time.Time      `json:"buildTime"`
	AbiInfo               string         `json:"abiInfo"`
	ByteCode              string         `json:"byteCode"`
	AptosMv               string         `json:"aptosMv" gorm:"column:aptos_mv"`
	SolanaContractPrivkey string         `json:"solanaContractPrivkey" gorm:"column:solana_contract_privkey"`
	CreateTime            time.Time      `gorm:"column:create_time;default:current_timestamp" json:"createTime"`
	Type                  uint           `json:"type"`   // see #consts.ProjectFrameType
	Status                uint           `json:"status"` // 1: deploying, 2: success , 3: fail
	Branch                string         `json:"branch"`
	CommitId              string         `json:"commitId"`
	CommitInfo            string         `json:"commitInfo"`
	CodeInfo              string         `json:"codeInfo"`
}

type ContractDeploy struct {
	Id            uint      `gorm:"primaryKey" json:"id"`
	ContractId    uint      `json:"contractId"`
	ProjectId     uuid.UUID `json:"projectId"`
	Version       string    `json:"version"`
	DeployTime    time.Time `gorm:"column:deploy_time;default:current_timestamp" json:"deployTime"`
	Network       string    `json:"network"`
	Address       string    `json:"address"`
	CreateTime    time.Time `gorm:"column:create_time;default:current_timestamp" json:"createTime"`
	Type          uint      `json:"type"` // see #consts.ProjectFrameType
	DeclareTxHash string    `json:"declareTxHash"`
	DeployTxHash  string    `json:"deployTxHash"`
	Status        uint      `json:"status"` // 1: deploying, 2: success , 3: fail
	AbiInfo       string    `json:"abiInfo"`
	Branch        string    `json:"branch"`
	CommitId      string    `json:"commitId"`
	CommitInfo    string    `json:"commitInfo"`
}
