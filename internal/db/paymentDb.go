package db

import (
	"container/list"
	"context"
	"errors"
	"fmt"
	"time"
)

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
	DaysSubscribe int    `json:"days_subscribe"`
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

func (db *DB) AddPayment(groupId, userId, currencyId, amount int, groupIdTg, userIdTg, currency, invoicePayload, description, paymentMethod string, productId int) error {

	pgTag, err := db.conn.Exec(context.Background(),
		"INSERT INTO payment (group_id, group_id_tg, user_id, user_id_tg, currency_id, amount, currency, invoice_payload, description, payment_method, product_id) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)",
		groupId, groupIdTg, userId, userIdTg, currencyId, amount, currency, invoicePayload, description, paymentMethod, productId)
	if err != nil {
		return errors.New(fmt.Sprintf("PgTag: %s, err: %s", pgTag, err.Error()))
	}
	return nil
}

func (db *DB) GetPaymentStatus(payload string) (bool, error) {
	var is_paid = false
	err := db.conn.QueryRow(context.Background(), "SELECT is_paid FROM payment WHERE invoice_payload=$1", payload).Scan(&is_paid)
	return is_paid, err
}

func (db *DB) UpdateSuccessfulPayment(payload, providerPaymentChargeId, telegramPaymentChargeId string, is_paid, is_canceled bool, paid_at time.Time) error {

	if payload == "" {
		return errors.New("payload is empty")
	}
	_, err := db.conn.Exec(context.Background(),
		"UPDATE payment SET provider_payment_charge_id=$1, telegram_payment_charge_id=$2, is_paid=$3, is_canceled=$4, paid_at=$5 WHERE invoice_payload=$6", providerPaymentChargeId, telegramPaymentChargeId, is_paid, is_canceled, paid_at, payload)

	return err
}

func (db *DB) GetProductData(payload string) (Product, error) {
	var product Product
	err := db.conn.QueryRow(context.Background(),
		"SELECT p.id, c.id, p.name, p.descr, pop.price, c.name, pd.token, p.days_sub "+
			"FROM product p JOIN price_of_product pop ON p.id = pop.product_id "+
			"JOIN currency c ON pop.currency_id = c.id JOIN my_provider pd ON c.provider_id=pd.id "+
			"JOIN payment pt ON pt.product_id=p.id WHERE pt.invoice_payload=$1", payload).Scan(
		&product.ProductId, &product.CurrencyId, &product.Name, &product.Description, &product.Price, &product.CurrencyName,
		&product.ProviderToken, &product.DaysSubscribe)

	return product, err
}

func (db *DB) GetPaymentDataFromInvoice(payload string) (PaymentData, error) {
	var paymentData PaymentData
	err := db.conn.QueryRow(context.Background(),
		"SELECT group_id_tg, product_id, currency_id FROM payment WHERE invoice_payload=$1", payload).Scan(
		&paymentData.GroupId, &paymentData.ProductId, &paymentData.CurrencyId)
	return paymentData, err
}
