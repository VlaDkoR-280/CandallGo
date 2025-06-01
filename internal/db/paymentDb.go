package db

import (
	"container/list"
	"context"
	"errors"
	"fmt"
	"time"
)

func (db *DB) PaymentGetProviders() {

}

type Currency struct {
	Id         int    `json:"id"`
	Name       string `json:"name"`
	ProviderId int    `json:"provider_id"`
}

func (db *DB) GetCurrency(currencies *list.List) error {
	rows, err := db.conn.Query(context.Background(),
		"SELECT id, name, provider_id FROM currency")
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var currency Currency
		if err := rows.Scan(&currency.Id, &currency.Name, &currency.ProviderId); err != nil {
			return err
		}
		currencies.PushBack(currency)
	}
	return nil
}

type Product struct {
	ProductId     int    `json:"product_id"`
	CurrencyId    int    `json:"currency_id"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	Price         int64  `json:"price"`
	CurrencyName  string `json:"currency_name"`
	ProviderToken string `json:"provider_token"`
}

type PaymentData struct {
	GroupId    string `json:"group_id"`
	ProductId  int    `json:"product_id"`
	CurrencyId int    `json:"currency_id"`
}

func (db *DB) GetPrices(productList *list.List) error {
	rows, err := db.conn.Query(context.Background(),
		"SELECT p.id, c.id, p.name, p.descr, pop.price, c.name, pd.token FROM product p "+
			"JOIN price_of_product pop ON p.id = pop.product_id "+
			"JOIN currency c ON pop.currency_id = c.id "+
			"JOIN my_provider pd ON c.provider_id=pd.id "+
			"WHERE pd.is_active = false")
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var product Product
		if err := rows.Scan(&product.ProductId, &product.CurrencyId,
			&product.Name, &product.Description, &product.Price, &product.CurrencyName, &product.ProviderToken); err != nil {
			return err
		}
		productList.PushBack(product)
	}
	return nil

}

func (db *DB) AddPayment(groupId, userId, currencyId, amount int, groupIdTg, userIdTg, currency, invoicePayload, description, paymentMethod string) error {

	pgTag, err := db.conn.Exec(context.Background(),
		"INSERT INTO payment (group_id, group_id_tg, user_id, user_id_tg, currency_id, amount, currency, invoice_payload, description, payment_method) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)",
		groupId, groupIdTg, userId, userIdTg, currencyId, amount, currency, invoicePayload, description, paymentMethod)
	if err != nil {
		return errors.New(fmt.Sprintf("PgTag: %s, err: %s", pgTag, err.Error()))
	}
	return nil
}

func (db *DB) UpdateSuccessfulPayment(payload string, providerPaymentChargeId, telegramPaymentChargeId int, is_paid, is_canceled bool, paid_at time.Time) error {
	return nil
}
