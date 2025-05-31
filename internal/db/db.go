package db

import (
	"container/list"
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
)

type DB struct {
	conn *pgxpool.Pool
}

type UserData struct {
	Id   int
	TgId string
}

type GroupData struct {
	Id             int
	TgId           string
	GroupName      string
	IsGroup        bool
	DateLastUse    time.Time
	SubDateEnd     time.Time
	TimeLastUpdate time.Time
}

func Connect(databaseUrl string) (*DB, error) {
	pool, err := pgxpool.New(context.Background(), databaseUrl)
	return &DB{pool}, err
}

func (db *DB) Close() {
	db.conn.Close()
}

// GetUserData получение пользователя UserData
func (db *DB) GetUserData(userId string) (userData UserData, err error) {
	err = db.conn.QueryRow(context.Background(), "SELECT id, tg_id FROM user_data WHERE tg_id=$1", userId).Scan(&userData.Id, &userData.TgId)
	return
}

// GetGroupData получение GroupData
func (db *DB) GetGroupData(groupId string) (groupData GroupData, err error) {
	err = db.conn.QueryRow(context.Background(), "SELECT id, tg_id, name, isgroup, data_last_use, sub_end_date, time_last_update FROM group_data WHERE tg_id=$1", groupId).Scan(
		&groupData.Id, &groupData.TgId, &groupData.GroupName, &groupData.IsGroup, &groupData.DateLastUse, &groupData.SubDateEnd, &groupData.TimeLastUpdate)
	if err != nil {
		return GroupData{}, err
	}
	return
}

// GetGroupsOfUser получение списка групп пользователя
func (db *DB) GetGroupsOfUser(userId string, groupList *list.List, onlyGroup bool) error {
	rows, err := db.conn.Query(context.Background(), "SELECT g.id, g.tg_id, g.name, g.isgroup, g.data_last_use, g.sub_end_date, g.time_last_update FROM group_data g JOIN users_of_group ug ON ug.group_id=g.id JOIN user_data u ON u.id=ug.user_id WHERE u.tg_id=$1", userId)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var group GroupData
		err := rows.Scan(&group.Id, &group.TgId, &group.GroupName, &group.IsGroup, &group.DateLastUse, &group.SubDateEnd, &group.TimeLastUpdate)
		if err != nil {
			return err
		}
		if group.IsGroup || !onlyGroup {
			groupList.PushBack(group)
		}
	}

	return nil
}

// GetUsersFromGroup получение списка пользователей в группе
func (db *DB) GetUsersFromGroup(groupId string, userList *list.List) error {

	rows, err := db.conn.Query(context.Background(), "SELECT u.id, u.tg_id FROM user_data u JOIN users_of_group ug ON ug.user_id=u.id JOIN group_data g ON g.id=ug.group_id WHERE g.tg_id=$1", groupId)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var userData UserData
		err := rows.Scan(&userData.Id, &userData.TgId)
		if err != nil {
			return err
		}
		userList.PushBack(userData)
	}
	return nil
}
