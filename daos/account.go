package daos

import (
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/Proofsuite/amp-matching-engine/app"
	"github.com/Proofsuite/amp-matching-engine/types"
	"github.com/ethereum/go-ethereum/common"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// BalanceDao contains:
// collectionName: MongoDB collection name
// dbName: name of mongodb to interact with
type AccountDao struct {
	collectionName string
	dbName         string
}

// NewBalanceDao returns a new instance of AddressDao
func NewAccountDao() *AccountDao {
	dbName := app.Config.DBName
	collection := "accounts"
	index := mgo.Index{
		Key:    []string{"address"},
		Unique: true,
	}

	err := db.Session.DB(dbName).C(collection).EnsureIndex(index)
	if err != nil {
		panic(err)
	}

	return &AccountDao{collection, dbName}
}

// Create function performs the DB insertion task for Balance collection
func (dao *AccountDao) Create(account *types.Account) error {
	account.ID = bson.NewObjectId()
	account.CreatedAt = time.Now()
	account.UpdatedAt = time.Now()

	err := db.Create(dao.dbName, dao.collectionName, account)
	if err != nil {
		log.Print(err)
		return err
	}

	return nil
}

func (dao *AccountDao) GetAll() (res []types.Account, err error) {
	err = db.Get(dao.dbName, dao.collectionName, bson.M{}, 0, 0, &res)
	return
}

func (dao *AccountDao) GetByID(id bson.ObjectId) (*types.Account, error) {
	res := []types.Account{}
	q := bson.M{"_id": id}

	err := db.Get(dao.dbName, dao.collectionName, q, 0, 0, &res)
	if err != nil {
		return nil, err
	}

	return &res[0], nil
}

func (dao *AccountDao) GetByAddress(owner common.Address) (response *types.Account, err error) {
	var res []*types.Account
	q := bson.M{"address": owner.Hex()}
	err = db.Get(dao.dbName, dao.collectionName, q, 0, 1, &res)

	if err != nil {
		return
	} else if len(res) > 0 {
		response = res[0]
		return
	}

	return nil, fmt.Errorf("NO_ACCOUNT_FOUND")
}

func (dao *AccountDao) GetTokenBalances(owner common.Address) (map[common.Address]*types.TokenBalance, error) {
	q := bson.M{"address": owner.Hex()}
	response := []types.Account{}
	err := db.Get(dao.dbName, dao.collectionName, q, 0, 1, &response)
	if err != nil {
		return nil, err
	}

	if len(response) > 0 {
		return response[0].TokenBalances, nil
	}

	return nil, fmt.Errorf("NO_ACCOUNT_FOUND")
}

func (dao *AccountDao) GetTokenBalance(owner common.Address, token common.Address) (*types.TokenBalance, error) {
	q := []bson.M{
		bson.M{
			"$match": bson.M{
				"address": owner.Hex(),
			},
		},
		bson.M{
			"$project": bson.M{
				"tokenBalances": bson.M{
					"$objectToArray": "$tokenBalances",
				},
				"_id": 0,
			},
		},
		bson.M{
			"$addFields": bson.M{
				"tokenBalances": bson.M{
					"$filter": bson.M{
						"input": "$tokenBalances",
						"as":    "kv",
						"cond": bson.M{
							"$eq": []interface{}{"$$kv.k", token.Hex()},
						},
					},
				},
			},
		},
		bson.M{
			"$addFields": bson.M{
				"tokenBalances": bson.M{
					"$arrayToObject": "$tokenBalances",
				},
			},
		},
	}

	var res []*types.Account
	err := db.Aggregate(dao.dbName, dao.collectionName, q, &res)
	if err != nil {
		return nil, err
	}
	//
	//a := &types.Account{}
	//bytes, _ := bson.Marshal(res[0])
	//bson.Unmarshal(bytes, &a)

	return res[0].TokenBalances[token], nil
}

func (dao *AccountDao) UpdateTokenBalance(owner, token common.Address, tokenBalance *types.TokenBalance) error {
	q := bson.M{
		"address": owner.Hex(),
	}

	updateQuery := bson.M{
		"$set": bson.M{
			"tokenBalances." + token.Hex() + ".balance":        tokenBalance.Balance.String(),
			"tokenBalances." + token.Hex() + ".allowance":      tokenBalance.Allowance.String(),
			"tokenBalances." + token.Hex() + ".lockedBalance":  tokenBalance.LockedBalance.String(),
			"tokenBalances." + token.Hex() + ".pendingBalance": tokenBalance.PendingBalance.String(),
		},
	}

	err := db.Update(dao.dbName, dao.collectionName, q, updateQuery)
	return err
}

func (dao *AccountDao) UpdateBalance(owner common.Address, token common.Address, balance *big.Int) error {
	q := bson.M{
		"address": owner.Hex(),
	}
	updateQuery := bson.M{
		"$set": bson.M{"tokenBalances." + token.Hex() + ".balance": balance.String()},
	}

	err := db.Update(dao.dbName, dao.collectionName, q, updateQuery)
	return err
}

func (dao *AccountDao) UpdateAllowance(owner common.Address, token common.Address, allowance *big.Int) error {
	q := bson.M{
		"address": owner.Hex(),
	}
	updateQuery := bson.M{
		"$set": bson.M{"tokenBalances." + token.Hex() + ".allowance": allowance.String()},
	}

	err := db.Update(dao.dbName, dao.collectionName, q, updateQuery)
	return err
}

// Drop drops all the order documents in the current database
func (dao *AccountDao) Drop() {
	db.DropCollection(dao.dbName, dao.collectionName)
}
