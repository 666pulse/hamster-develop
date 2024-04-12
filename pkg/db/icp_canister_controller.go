package db

import "database/sql"

type IcpCanisterController struct {
	Id         int          `json:"id"`
	FkUserId   uint         `json:"fkUserId"`
	CanisterId string       `json:"canisterId"`
	Controller string       `json:"controller"`
	CreateTime sql.NullTime `json:"createTime"`
}
