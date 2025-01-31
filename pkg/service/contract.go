package service

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dontpanicdao/caigo/gateway"
	"github.com/dontpanicdao/caigo/types"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/goperate/convert/core/array"
	"github.com/hamster-shared/aline-engine/logger"
	"github.com/hamster-shared/hamster-develop/pkg/application"
	"github.com/hamster-shared/hamster-develop/pkg/consts"
	db2 "github.com/hamster-shared/hamster-develop/pkg/db"
	"github.com/hamster-shared/hamster-develop/pkg/parameter"
	"github.com/hamster-shared/hamster-develop/pkg/utils"
	"github.com/hamster-shared/hamster-develop/pkg/vo"
	uuid "github.com/iris-contrib/go.uuid"
	"github.com/jinzhu/copier"
	"gorm.io/gorm"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type ContractService struct {
	db *gorm.DB
	gw *gateway.Gateway
}

func NewContractService() *ContractService {
	gw := gateway.NewClient(gateway.WithChain(gateway.GOERLI_ID))
	return &ContractService{
		db: application.GetBean[*gorm.DB]("db"),
		gw: gw,
	}
}

type DeclareRequest struct {
	Type          string              `json:"type"`
	SenderAddress string              `json:"sender_address"`
	MaxFee        string              `json:"max_fee"`
	Nonce         string              `json:"nonce"`
	Signature     []string            `json:"signature"`
	ContractClass types.ContractClass `json:"contract_class"`
	Version       string              `json:"version"`
}

func (c *ContractService) Declare(gw *gateway.Gateway, ctx context.Context, contract types.ContractClass) (resp types.AddDeclareResponse, err error) {
	declareRequest := DeclareRequest{}
	declareRequest.Type = "DECLARE"
	declareRequest.SenderAddress = "0x1"
	declareRequest.MaxFee = "0x0"
	declareRequest.Nonce = "0x0"
	declareRequest.Signature = []string{}
	declareRequest.ContractClass = contract
	declareRequest.Version = "0x0"
	req, err := c.newRequest(gw, ctx, http.MethodPost, "/add_transaction", declareRequest)
	if err != nil {
		return resp, err
	}

	return resp, c.do(req, &resp)

}

func (c *ContractService) newRequest(
	gw *gateway.Gateway, ctx context.Context, method, endpoint string, body interface{},
) (*http.Request, error) {
	url := gw.Feeder + endpoint
	if strings.HasSuffix(endpoint, "add_transaction") {
		url = gw.Gateway + endpoint
	}

	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, err
	}

	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal body: %w", err)
		}
		req.Body = io.NopCloser(bytes.NewBuffer(data))
		req.Header.Add("Content-Type", "application/json; charset=utf")
	}
	return req, nil
}

func (c *ContractService) do(req *http.Request, v interface{}) error {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close() // nolint: errcheck

	if resp.StatusCode >= 299 {
		e := NewError(resp)
		return e
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, v)
}

type Error struct {
	StatusCode int    `json:"-"`
	Body       []byte `json:"-"`

	Code    string `json:"code"`
	Message string `json:"message"`
}

// Error implements the error interface.
func (e Error) Error() string {
	return fmt.Sprintf("%d: %s %s", e.StatusCode, e.Code, e.Message)
}

func NewError(resp *http.Response) error {
	apiErr := Error{StatusCode: resp.StatusCode}
	data, err := io.ReadAll(resp.Body)
	if err == nil && data != nil {
		apiErr.Body = data
		if err := json.Unmarshal(data, &apiErr); err != nil {
			apiErr.Code = "unknown_error_format"
			apiErr.Message = string(data)
		}
	}
	return &apiErr
}

func (c *ContractService) DoStarknetDeclare(compiledContract []byte) (txHash string, classHash string, err error) {
	gw := gateway.NewClient(gateway.WithChain(gateway.GOERLI_ID))
	ctx := context.Background()
	var contractClass types.ContractClass
	err = json.Unmarshal(compiledContract, &contractClass)
	if err != nil {
		return "", "", err
	}
	declare, err := c.Declare(gw, ctx, contractClass)
	if err != nil {
		return "", "", err
	}
	fmt.Println("declare.TransactionHash: ", declare.TransactionHash)
	fmt.Println("declare.ClassHash: ", declare.ClassHash)

	_, receipt, err := gw.WaitForTransaction(ctx, declare.TransactionHash, 3, 10)
	if err != nil {
		fmt.Printf("could not declare contract: %v\n", err)
		return "", "", err
	}
	if receipt.Status != types.TransactionAcceptedOnL1 && receipt.Status != types.TransactionAcceptedOnL2 {
		fmt.Printf("unexpected status: %s\n", receipt.Status)
		return "", "", err
	}

	return declare.TransactionHash, declare.ClassHash, nil
}

//func (c *ContractService) DoStarknetDeclare(compiledContract []byte) (txHash string, classHash string, err error) {
//	gw := gateway.NewClient(gateway.WithChain(gateway.GOERLI_ID))
//
//	ctx := context.Background()
//	var contractClass types.ContractClass
//
//	err = json.Unmarshal(compiledContract, &contractClass)
//
//	if err != nil {
//		return "", "", err
//	}
//
//	declare, err := gw.Declare(ctx, contractClass, gateway.DeclareRequest{})
//	if err != nil {
//		return "", "", err
//	}
//
//	fmt.Println("declare.TransactionHash: ", declare.TransactionHash)
//	fmt.Println("declare.ClassHash: ", declare.ClassHash)
//
//	_, receipt, err := gw.WaitForTransaction(ctx, declare.TransactionHash, 3, 10)
//	if err != nil {
//		fmt.Printf("could not declare contract: %v\n", err)
//		return "", "", err
//	}
//	if receipt.Status != types.TransactionAcceptedOnL1 && receipt.Status != types.TransactionAcceptedOnL2 {
//		fmt.Printf("unexpected status: %s\n", receipt.Status)
//		return "", "", err
//	}
//
//	return declare.TransactionHash, declare.ClassHash, nil
//}

func (c *ContractService) SaveDeploy(deployParam parameter.ContractDeployParam) (uint, error) {
	var entity db2.ContractDeploy
	_ = c.db.Model(&db2.ContractDeploy{}).Where("deploy_tx_hash = ?", deployParam.DeployTxHash).First(&entity).Error
	projectId, err := uuid.FromString(deployParam.ProjectId)
	if err != nil {
		return 0, err
	}

	projectService := application.GetBean[*ProjectService]("projectService")
	project, err := projectService.GetProject(projectId.String(), 0)
	_ = copier.Copy(&entity, &deployParam)
	entity.DeployTime = time.Now()
	entity.ProjectId = projectId

	if project.FrameType == consts.Evm {
		entity.Status = consts.STATUS_SUCCESS
	}

	var contract db2.Contract
	err = c.db.Model(db2.Contract{}).Where("id = ?", entity.ContractId).First(&contract).Error
	if err != nil {
		return 0, err
	}
	version, err := strconv.Atoi(entity.Version)
	if err != nil {
		return 0, err
	}
	entity.Type = contract.Type
	if entity.AbiInfo == "" && version > 1 {
		for {
			if version > 1 {
				var contractDeploy db2.ContractDeploy
				c.db.Model(db2.ContractDeploy{}).Where("contract_id = ? and version = ? ", entity.ContractId, entity.Version).First(&contractDeploy)
				if contractDeploy.AbiInfo != "" {
					entity.AbiInfo = contractDeploy.AbiInfo
					break
				}
				version = version - 1
			} else {
				break
			}
		}
	}
	if contract.AbiInfo == "" {
		contract.AbiInfo = entity.AbiInfo
	}
	err = c.db.Save(&entity).Error
	if err != nil {
		return 0, err
	}
	network := entity.Network
	if contract.Network.String != "" {
		network = networkDistinct(contract.Network.String, entity.Network)
	}
	contract.Network = sql.NullString{
		String: network,
		Valid:  true,
	}
	contract.Status = entity.Status
	c.db.Save(&contract)
	return entity.Id, err
}

func (c *ContractService) QueryContracts(projectId string, query, version, network string, page int, size int) (vo.Page[vo.ContractArtifactsVo], error) {
	var contracts []db2.Contract
	var afterData []db2.Contract
	var project db2.Project
	err := c.db.Model(&db2.Project{}).Where("id = ?", projectId).First(&project).Error
	if err != nil {
		return vo.Page[vo.ContractArtifactsVo]{}, err
	}
	if project.FrameType == consts.InternetComputer {
		return c.QueryContractsForICP(projectId, query, version, network, page, size)
	}

	sql := fmt.Sprintf("select id, project_id,workflow_id,workflow_detail_id,name,version,group_concat( DISTINCT `network` SEPARATOR ',' ) as network,build_time,abi_info,byte_code,create_time,branch,commit_id,commit_info from t_contract where project_id = ? ")
	if query != "" && version != "" && network != "" {
		sql = sql + "and name like CONCAT('%',?,'%') and version = ? and network like CONCAT('%',?,'%') group by id order by create_time desc"
		c.db.Raw(sql, projectId, query, version, network).Scan(&contracts)
	} else if query != "" && version != "" {
		sql = sql + "and name like CONCAT('%',?,'%') and version = ? group by id order by create_time desc"
		c.db.Raw(sql, projectId, query, version).Scan(&contracts)
	} else if query != "" && network != "" {
		sql = sql + "and name like CONCAT('%',?,'%') and network like CONCAT('%',?,'%') group by id order by create_time desc"
		c.db.Raw(sql, projectId, query, network).Scan(&contracts)
	} else if version != "" && network != "" {
		sql = sql + "and version = ? and network like CONCAT('%',?,'%') group by id order by create_time desc"
		c.db.Raw(sql, projectId, version, network).Scan(&contracts)
	} else if query != "" {
		sql = sql + "and name like CONCAT('%',?,'%') group by id order by create_time desc"
		c.db.Raw(sql, projectId, query).Scan(&contracts)
	} else if network != "" {
		sql = sql + "and network like CONCAT('%',?,'%') group by id order by create_time desc"
		c.db.Raw(sql, projectId, network).Scan(&contracts)
	} else if version != "" {
		sql = sql + "and version = ? group by id order by create_time desc"
		c.db.Raw(sql, projectId, version).Scan(&contracts)
	} else {
		sql = sql + "group by id order by create_time desc"
		c.db.Raw(sql, projectId).Scan(&contracts)
	}
	if len(contracts) > 0 {
		start, end := utils.SlicePage(int64(page), int64(size), int64(len(contracts)))
		afterData = contracts[start:end]
	}
	var contractIdList []uint
	for _, contract := range afterData {
		contractIdList = append(contractIdList, contract.Id)
	}
	var contractDeployList []db2.ContractDeploy
	err = c.db.Table("(?) as u", c.db.Model(db2.ContractDeploy{}).Where("contract_id in ?", contractIdList).Order("deploy_time desc").Limit(1000)).Group("contract_id").Find(&contractDeployList).Error
	if err != nil {
		return vo.Page[vo.ContractArtifactsVo]{}, err
	}
	contractIdContractDeployIdyMap := make(map[uint]uint)
	for _, contractDeploy := range contractDeployList {
		contractIdContractDeployIdyMap[contractDeploy.ContractId] = contractDeploy.Id
	}
	var contractArtifactsVoList []vo.ContractArtifactsVo
	for _, contract := range afterData {
		var contractArtifactsVo vo.ContractArtifactsVo
		copier.Copy(&contractArtifactsVo, &contract)
		contractArtifactsVo.LastContractDeployId = contractIdContractDeployIdyMap[contract.Id]
		contractArtifactsVoList = append(contractArtifactsVoList, contractArtifactsVo)
	}
	return vo.NewPage[vo.ContractArtifactsVo](contractArtifactsVoList, len(contracts), page, size), nil
}

func (c *ContractService) QueryContractsForICP(projectId string, query, version, network string, page int, size int) (vo.Page[vo.ContractArtifactsVo], error) {
	var backendPackages []db2.BackendPackage
	var afterData []db2.BackendPackage
	sql := fmt.Sprintf("select id, project_id,workflow_id,workflow_detail_id,name,version,group_concat( DISTINCT `network` SEPARATOR ',' ) as network,build_time,abi_info,create_time,branch,commit_id,commit_info from t_backend_package where project_id = ? ")
	if query != "" && version != "" && network != "" {
		sql = sql + "and name like CONCAT('%',?,'%') and version = ? and network like CONCAT('%',?,'%') group by id order by create_time desc"
		c.db.Raw(sql, projectId, query, version, network).Scan(&backendPackages)
	} else if query != "" && version != "" {
		sql = sql + "and name like CONCAT('%',?,'%') and version = ? group by id order by create_time desc"
		c.db.Raw(sql, projectId, query, version).Scan(&backendPackages)
	} else if query != "" && network != "" {
		sql = sql + "and name like CONCAT('%',?,'%') and network like CONCAT('%',?,'%') group by id order by create_time desc"
		c.db.Raw(sql, projectId, query, network).Scan(&backendPackages)
	} else if version != "" && network != "" {
		sql = sql + "and version = ? and network like CONCAT('%',?,'%') group by id order by create_time desc"
		c.db.Raw(sql, projectId, version, network).Scan(&backendPackages)
	} else if query != "" {
		sql = sql + "and name like CONCAT('%',?,'%') group by id order by create_time desc"
		c.db.Raw(sql, projectId, query).Scan(&backendPackages)
	} else if network != "" {
		sql = sql + "and network like CONCAT('%',?,'%') group by id order by create_time desc"
		c.db.Raw(sql, projectId, network).Scan(&backendPackages)
	} else if version != "" {
		sql = sql + "and version = ? group by id order by create_time desc"
		c.db.Raw(sql, projectId, version).Scan(&backendPackages)
	} else {
		sql = sql + "group by id order by create_time desc"
		c.db.Raw(sql, projectId).Scan(&backendPackages)
	}
	if len(backendPackages) > 0 {
		start, end := utils.SlicePage(int64(page), int64(size), int64(len(backendPackages)))
		afterData = backendPackages[start:end]
	}
	var packageIdList []uint
	for _, backendPackage := range afterData {
		packageIdList = append(packageIdList, backendPackage.Id)
	}
	var backendDeployList []db2.BackendDeploy
	err := c.db.Table("(?) as u", c.db.Model(db2.BackendDeploy{}).Where("package_id in ?", packageIdList).Order("deploy_time desc").Limit(1000)).Group("package_id").Find(&backendDeployList).Error
	if err != nil {
		return vo.Page[vo.ContractArtifactsVo]{}, err
	}
	contractIdContractDeployIdyMap := make(map[uint]uint)
	for _, backendDeploy := range backendDeployList {
		contractIdContractDeployIdyMap[backendDeploy.PackageId] = backendDeploy.Id
	}
	var contractArtifactsVoList []vo.ContractArtifactsVo
	for _, backendPackage := range afterData {
		var contractArtifactsVo vo.ContractArtifactsVo
		copier.Copy(&contractArtifactsVo, &backendPackage)
		contractArtifactsVo.LastContractDeployId = contractIdContractDeployIdyMap[backendPackage.Id]
		contractArtifactsVoList = append(contractArtifactsVoList, contractArtifactsVo)
	}
	return vo.NewPage[vo.ContractArtifactsVo](contractArtifactsVoList, len(backendPackages), page, size), nil
}

func (c *ContractService) QueryContractByWorkflow(id string, workflowId, workflowDetailId int) ([]db2.Contract, error) {
	var project db2.Project
	var contracts []db2.Contract
	err := c.db.Where("id = ? ", id).First(&project).Error
	if err != nil {
		return contracts, err
	}
	if project.FrameType == consts.InternetComputer {
		var backendPackages []db2.BackendPackage
		err = c.db.Model(db2.BackendPackage{}).Where("workflow_id = ? and workflow_detail_id = ?", workflowId, workflowDetailId).Order("version DESC").Find(&backendPackages).Error
		if err != nil {
			return contracts, err
		}
		_ = copier.Copy(&contracts, &backendPackages)
	} else {
		res := c.db.Model(db2.Contract{}).Where("workflow_id = ? and workflow_detail_id = ?", workflowId, workflowDetailId).Order("version DESC").Find(&contracts)
		if res != nil {
			return contracts, res.Error
		}
	}
	return contracts, nil
}

func (c *ContractService) QueryContractByVersion(projectId string, version string) ([]vo.ContractVo, error) {
	var contracts []db2.Contract
	var data []vo.ContractVo
	res := c.db.Model(db2.Contract{}).Where("project_id = ? and version = ?", projectId, version).Find(&contracts)
	if res.Error != nil {
		return data, res.Error
	}
	if len(contracts) > 0 {
		_ = copier.Copy(&data, &contracts)
	}
	return data, nil
}

func (c *ContractService) QueryContractDeployByVersion(projectId string, version string) (vo.ContractDeployInfoVo, error) {
	var data vo.ContractDeployInfoVo
	var project db2.Project
	err := c.db.Where("id = ? ", projectId).First(&project).Error
	if err != nil {
		return data, err
	}

	if project.FrameType == consts.InternetComputer {
		var backendDeployData []db2.BackendDeploy
		res := c.db.Model(db2.BackendDeploy{}).Where("project_id = ? and version = ?", projectId, version).Find(&backendDeployData)
		if res.Error != nil {
			return data, res.Error
		}
		contractInfo := make(map[string]vo.ContractInfoVo)
		if len(backendDeployData) > 0 {
			arr := array.NewObjArray(backendDeployData, "PackageId")
			res2 := arr.ToIdMapArray().(map[uint][]db2.BackendDeploy)
			for u, deploys := range res2 {
				var backendPackageData db2.BackendPackage
				res := c.db.Model(db2.BackendPackage{}).Where("id = ?", u).First(&backendPackageData)
				var contractInfoVo vo.ContractInfoVo
				copier.Copy(&contractInfoVo, &backendPackageData)
				if res.Error == nil {
					var deployInfo []vo.DeployInfVo
					if len(deploys) > 0 {
						for _, deploy := range deploys {
							var deployData vo.DeployInfVo
							copier.Copy(&deployData, &deploy)
							deployInfo = append(deployInfo, deployData)
							if deploy.AbiInfo != "" && contractInfoVo.AbiInfo == "" {
								contractInfoVo.AbiInfo = deploy.AbiInfo
							}
						}
					}
					contractInfoVo.DeployInfo = deployInfo
					contractInfo[backendPackageData.Name] = contractInfoVo
				}
			}
		}
		data.Version = version
		data.ContractInfo = contractInfo
	} else {
		var contractDeployData []db2.ContractDeploy
		res := c.db.Model(db2.ContractDeploy{}).Where("project_id = ? and version = ?", projectId, version).Find(&contractDeployData)
		if res.Error != nil {
			return data, res.Error
		}
		contractInfo := make(map[string]vo.ContractInfoVo)
		if len(contractDeployData) > 0 {
			arr := array.NewObjArray(contractDeployData, "ContractId")
			res2 := arr.ToIdMapArray().(map[uint][]db2.ContractDeploy)
			for u, deploys := range res2 {
				var contractData db2.Contract
				res := c.db.Model(db2.Contract{}).Where("id = ?", u).First(&contractData)
				var contractInfoVo vo.ContractInfoVo
				copier.Copy(&contractInfoVo, &contractData)
				if res.Error == nil {
					var deployInfo []vo.DeployInfVo
					if len(deploys) > 0 {
						for _, deploy := range deploys {
							var deployData vo.DeployInfVo
							copier.Copy(&deployData, &deploy)
							deployInfo = append(deployInfo, deployData)
							if deploy.AbiInfo != "" && contractInfoVo.AbiInfo == "" {
								contractInfoVo.AbiInfo = deploy.AbiInfo
							}
						}
					}
					contractInfoVo.DeployInfo = deployInfo
					contractInfo[contractData.Name] = contractInfoVo
				}
			}
		}
		data.Version = version
		data.ContractInfo = contractInfo
	}
	return data, nil
}

func (c *ContractService) QueryVersionList(projectId string) ([]string, error) {
	var data []string
	var project db2.Project
	err := c.db.Where("id = ? ", projectId).First(&project).Error
	if err != nil {
		return data, err
	}

	if project.FrameType == consts.InternetComputer {
		res := c.db.Model(db2.BackendPackage{}).Distinct("version").Select("version").Where("project_id = ?", projectId).Order("create_time desc").Find(&data)
		if res.Error != nil {
			return data, res.Error
		}
	} else {
		res := c.db.Model(db2.Contract{}).Distinct("version").Select("version").Where("project_id = ?", projectId).Order("create_time desc").Find(&data)
		if res.Error != nil {
			return data, res.Error
		}
	}

	return data, nil
}

func (c *ContractService) GetCodeInfoByVersion(projectId, version string) (vo.ContractVersionAndCodeInfoVo, error) {
	var contractVersionAndCodeInfoVo vo.ContractVersionAndCodeInfoVo
	var project db2.Project
	err := c.db.Where("id = ? ", projectId).First(&project).Error
	if err != nil {
		return contractVersionAndCodeInfoVo, err
	}

	if project.FrameType == consts.InternetComputer {
		res := c.db.Model(db2.BackendPackage{}).Select("version", "branch", "commit_id", "commit_info").Where("project_id = ? and version = ?", projectId, version).Order("create_time desc").Limit(1).First(&contractVersionAndCodeInfoVo)
		if res.Error != nil {
			return contractVersionAndCodeInfoVo, res.Error
		}
	} else {
		res := c.db.Model(db2.Contract{}).Select("version", "branch", "commit_id", "commit_info").Where("project_id = ? and version = ?", projectId, version).Order("create_time desc").Limit(1).First(&contractVersionAndCodeInfoVo)
		if res.Error != nil {
			return contractVersionAndCodeInfoVo, res.Error
		}
	}
	contractVersionAndCodeInfoVo.Type = int(project.Type)
	contractVersionAndCodeInfoVo.Url = project.RepositoryUrl
	return contractVersionAndCodeInfoVo, nil
}

func (c *ContractService) QueryContractNameList(projectId string) ([]string, error) {
	var data []string
	res := c.db.Model(db2.Contract{}).Distinct("name").Select("name").Where("project_id = ?", projectId).Find(&data)
	if res.Error != nil {
		return data, res.Error
	}
	return data, nil
}

func (c *ContractService) QueryNetworkList(projectId string) ([]string, error) {
	var data []string
	var result []string
	res := c.db.Model(db2.Contract{}).Distinct("network").Select("network").Where("project_id = ? and network != '' ", projectId).Find(&data)
	if res.Error != nil {
		return result, res.Error
	}
	longest := ""
	for _, str := range data {
		if len(str) > len(longest) {
			longest = str
		}
	}
	result = strings.Split(longest, ",")
	return result, nil
}

func (c *ContractService) SyncStarkWareContract() {

	var contractDeploys []db2.ContractDeploy

	const running = 1
	err := c.db.Model(db2.ContractDeploy{}).Where("type=? and status = ?", consts.StarkWare, running).Find(&contractDeploys).Error
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, deploy := range contractDeploys {

		if time.Since(deploy.DeployTime) > time.Minute*15 {
			// fail
			deploy.Status = 3
		} else if deploy.DeployTxHash != "" {
			receipt, err := c.gw.TransactionReceipt(context.Background(), deploy.DeployTxHash)
			if err != nil {
				continue
			}
			if receipt.Status == types.TransactionAcceptedOnL2 {
				// success

				// query contract address
				if len(receipt.Events) > 0 {
					event1 := receipt.Events[0].(map[string]interface{})
					data := event1["data"].([]interface{})
					if len(data) > 0 {
						contractAddress := data[0].(string)
						deploy.Address = contractAddress
						deploy.Status = 1

					}
				}
			}
		}
		err := c.db.Save(&deploy).Error
		if err != nil {
			fmt.Println("save contractDeploy error")
			continue
		}
		var contract db2.Contract
		err = c.db.Model(db2.Contract{}).Where("id = ?", deploy.ContractId).First(&contract).Error
		if err != nil {
			fmt.Println("save Contract error")
			continue
		}
		contract.Status = deploy.Status
		c.db.Save(contract.Status)
	}
}

func (c *ContractService) GetContractDeployInfo(id int) (vo.ContractDeployVo, error) {
	var result vo.ContractDeployVo
	var contractDeploy db2.ContractDeploy
	err := c.db.Model(db2.ContractDeploy{}).Where("id = ?", id).First(&contractDeploy).Error
	if err != nil {
		return result, err
	}
	var project db2.Project
	err = c.db.Model(db2.Project{}).Where("id = ?", contractDeploy.ProjectId).First(&project).Error
	if err != nil {
		return result, err
	}
	var contract db2.Contract
	err = c.db.Model(db2.Contract{}).Where("id = ?", contractDeploy.ContractId).First(&contract).Error
	if err != nil {
		return result, err
	}
	copier.Copy(&result, &contractDeploy)
	result.ContractName = contract.Name
	result.Url = project.RepositoryUrl
	result.Branch = contract.Branch
	result.CommitId = contract.CommitId
	result.CommitInfo = contract.CommitInfo
	return result, err
}

func (c *ContractService) SaveDeployIng(deployingParam parameter.ContractDeployIngParam) error {
	// check tx exists
	projectId, err := uuid.FromString(deployingParam.ProjectId)
	if err != nil {
		return err
	}
	go func() {
		receipt, transaction, err := getEthReceipt(deployingParam.RpcUrl, deployingParam.DeployTxHash)
		if err != nil {
			logger.Info("sync contract deploy fail: ", err)
			return
		}

		var deployInfo db2.ContractDeploy

		_ = c.db.Model(&db2.ContractDeploy{}).Where("deploy_tx_hash = ?", deployingParam.DeployTxHash).First(&deployInfo).Error

		deployInfo.ProjectId = projectId
		deployInfo.ContractId = deployingParam.ContractId
		deployInfo.Version = deployingParam.Version
		deployInfo.Network = deployingParam.Network
		deployInfo.DeployTime = transaction.Time()
		deployInfo.Address = receipt.ContractAddress.Hex()
		deployInfo.CreateTime = time.Now()
		deployInfo.Type = uint(consts.Evm)
		deployInfo.DeployTxHash = deployingParam.DeployTxHash
		deployInfo.Status = consts.STATUS_SUCCESS
		err = c.db.Save(&deployInfo).Error
		if err != nil {
			logger.Error("save db t_contract_deploy error : ", err)
		}
	}()

	return nil
}

func networkDistinct(oldNetwork, newNetwork string) string {
	arr := strings.Split(oldNetwork, ",")
	exist := false
	for _, network := range arr {
		if newNetwork == network {
			exist = true
			break
		}
	}
	if exist {
		return oldNetwork
	}
	return fmt.Sprintf("%s,%s", oldNetwork, newNetwork)
}

func getEthReceipt(rpcURL string, txHash string) (*ethtypes.Receipt, *ethtypes.Transaction, error) {
	// 连接以太坊节点
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		logger.Error(err)
		return nil, nil, err
	}

	// 你要查询的合约部署交易的哈希
	transactionHash := common.HexToHash(txHash)

	// 获取交易的详细信息
	transaction, _, err := client.TransactionByHash(context.Background(), transactionHash)
	if err != nil {
		logger.Error(err)
		return nil, nil, err
	}

	// 确认交易是合约创建交易
	if transaction.To() != nil {
		err = errors.New("Transaction is not a contract deployment")
		logger.Error(err)
		return nil, nil, err
	}
	// 等待合约部署成功
	receipt, err := waitForContractDeployment(client, transaction.Hash(), 15*time.Second)
	if err != nil {
		logger.Error(err)
		return nil, nil, err
	}

	fmt.Printf("Contract deployed at address: %s\n", receipt.ContractAddress.Hex())
	return receipt, transaction, err
}

// 等待合约部署成功
func waitForContractDeployment(client *ethclient.Client, transactionHash common.Hash, timeout time.Duration) (*ethtypes.Receipt, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// 创建一个通道来接收区块头

	for {
		select {
		case <-ctx.Done():
			fmt.Println("timeout")
			return nil, errors.New("timeout")
		default:
			receipt, err := client.TransactionReceipt(ctx, transactionHash)
			if err != nil {
				return nil, err
			}

			if receipt == nil {
				// 交易还未被打包，继续等待
				continue
			}

			// 如果交易被打包，并且合约地址不是空，返回交易收据
			if receipt.ContractAddress != common.HexToAddress("0xB362Eba0f3f42Ad32394f84ecb9c8d42bF1f2839") {
				return receipt, nil
			}

			time.Sleep(time.Second)
		}
	}
}
