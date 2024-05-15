// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.25.0
// source: gateways.sql

package sqlc

import (
	"context"
)

const addGateway = `-- name: AddGateway :exec
INSERT INTO gateways (name, endpoint, public_key, ipv4, ipv6, password_hash, requires_privileged_access)
VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7)
ON CONFLICT (name) DO
    UPDATE SET endpoint = excluded.endpoint, public_key = excluded.public_key, password_hash = excluded.password_hash, ipv6 = excluded.ipv6
`

type AddGatewayParams struct {
	Name                     string
	Endpoint                 string
	PublicKey                string
	Ipv4                     string
	Ipv6                     string
	PasswordHash             string
	RequiresPrivilegedAccess bool
}

func (q *Queries) AddGateway(ctx context.Context, arg AddGatewayParams) error {
	_, err := q.exec(ctx, q.addGatewayStmt, addGateway,
		arg.Name,
		arg.Endpoint,
		arg.PublicKey,
		arg.Ipv4,
		arg.Ipv6,
		arg.PasswordHash,
		arg.RequiresPrivilegedAccess,
	)
	return err
}

const addGatewayAccessGroupID = `-- name: AddGatewayAccessGroupID :exec
INSERT INTO gateway_access_group_ids (gateway_name, group_id)
VALUES (?1, ?2)
ON CONFLICT DO NOTHING
`

type AddGatewayAccessGroupIDParams struct {
	GatewayName string
	GroupID     string
}

func (q *Queries) AddGatewayAccessGroupID(ctx context.Context, arg AddGatewayAccessGroupIDParams) error {
	_, err := q.exec(ctx, q.addGatewayAccessGroupIDStmt, addGatewayAccessGroupID, arg.GatewayName, arg.GroupID)
	return err
}

const addGatewayRoute = `-- name: AddGatewayRoute :exec
INSERT INTO gateway_routes (gateway_name, route, family)
VALUES (?1, ?2, ?3)
ON CONFLICT DO NOTHING
`

type AddGatewayRouteParams struct {
	GatewayName string
	Route       string
	Family      string
}

func (q *Queries) AddGatewayRoute(ctx context.Context, arg AddGatewayRouteParams) error {
	_, err := q.exec(ctx, q.addGatewayRouteStmt, addGatewayRoute, arg.GatewayName, arg.Route, arg.Family)
	return err
}

const deleteGatewayAccessGroupIDs = `-- name: DeleteGatewayAccessGroupIDs :exec
DELETE FROM gateway_access_group_ids WHERE gateway_name = ?1
`

func (q *Queries) DeleteGatewayAccessGroupIDs(ctx context.Context, gatewayName string) error {
	_, err := q.exec(ctx, q.deleteGatewayAccessGroupIDsStmt, deleteGatewayAccessGroupIDs, gatewayName)
	return err
}

const deleteGatewayRoutes = `-- name: DeleteGatewayRoutes :exec
DELETE FROM gateway_routes WHERE gateway_name = ?1
`

func (q *Queries) DeleteGatewayRoutes(ctx context.Context, gatewayName string) error {
	_, err := q.exec(ctx, q.deleteGatewayRoutesStmt, deleteGatewayRoutes, gatewayName)
	return err
}

const getGatewayAccessGroupIDs = `-- name: GetGatewayAccessGroupIDs :many
SELECT group_id FROM gateway_access_group_ids WHERE gateway_name = ?1 ORDER BY group_id
`

func (q *Queries) GetGatewayAccessGroupIDs(ctx context.Context, gatewayName string) ([]string, error) {
	rows, err := q.query(ctx, q.getGatewayAccessGroupIDsStmt, getGatewayAccessGroupIDs, gatewayName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []string
	for rows.Next() {
		var group_id string
		if err := rows.Scan(&group_id); err != nil {
			return nil, err
		}
		items = append(items, group_id)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getGatewayByName = `-- name: GetGatewayByName :one
SELECT name, endpoint, public_key, ipv4, requires_privileged_access, password_hash, ipv6 FROM gateways WHERE name = ?1
`

func (q *Queries) GetGatewayByName(ctx context.Context, name string) (*Gateway, error) {
	row := q.queryRow(ctx, q.getGatewayByNameStmt, getGatewayByName, name)
	var i Gateway
	err := row.Scan(
		&i.Name,
		&i.Endpoint,
		&i.PublicKey,
		&i.Ipv4,
		&i.RequiresPrivilegedAccess,
		&i.PasswordHash,
		&i.Ipv6,
	)
	return &i, err
}

const getGatewayRoutes = `-- name: GetGatewayRoutes :many
SELECT route, family FROM gateway_routes WHERE gateway_name = ?1 ORDER BY route
`

type GetGatewayRoutesRow struct {
	Route  string
	Family string
}

func (q *Queries) GetGatewayRoutes(ctx context.Context, gatewayName string) ([]*GetGatewayRoutesRow, error) {
	rows, err := q.query(ctx, q.getGatewayRoutesStmt, getGatewayRoutes, gatewayName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*GetGatewayRoutesRow
	for rows.Next() {
		var i GetGatewayRoutesRow
		if err := rows.Scan(&i.Route, &i.Family); err != nil {
			return nil, err
		}
		items = append(items, &i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getGateways = `-- name: GetGateways :many
SELECT name, endpoint, public_key, ipv4, requires_privileged_access, password_hash, ipv6 FROM gateways ORDER BY name
`

func (q *Queries) GetGateways(ctx context.Context) ([]*Gateway, error) {
	rows, err := q.query(ctx, q.getGatewaysStmt, getGateways)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*Gateway
	for rows.Next() {
		var i Gateway
		if err := rows.Scan(
			&i.Name,
			&i.Endpoint,
			&i.PublicKey,
			&i.Ipv4,
			&i.RequiresPrivilegedAccess,
			&i.PasswordHash,
			&i.Ipv6,
		); err != nil {
			return nil, err
		}
		items = append(items, &i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const updateGateway = `-- name: UpdateGateway :exec
UPDATE gateways
SET public_key = ?1, endpoint = ?2, ipv4 = ?3, ipv6 = ?4, requires_privileged_access = ?5, password_hash = ?6
WHERE name = ?7
`

type UpdateGatewayParams struct {
	PublicKey                string
	Endpoint                 string
	Ipv4                     string
	Ipv6                     string
	RequiresPrivilegedAccess bool
	PasswordHash             string
	Name                     string
}

func (q *Queries) UpdateGateway(ctx context.Context, arg UpdateGatewayParams) error {
	_, err := q.exec(ctx, q.updateGatewayStmt, updateGateway,
		arg.PublicKey,
		arg.Endpoint,
		arg.Ipv4,
		arg.Ipv6,
		arg.RequiresPrivilegedAccess,
		arg.PasswordHash,
		arg.Name,
	)
	return err
}

const updateGatewayDynamicFields = `-- name: UpdateGatewayDynamicFields :exec
UPDATE gateways
SET requires_privileged_access = ?1
WHERE name = ?2
`

type UpdateGatewayDynamicFieldsParams struct {
	RequiresPrivilegedAccess bool
	Name                     string
}

func (q *Queries) UpdateGatewayDynamicFields(ctx context.Context, arg UpdateGatewayDynamicFieldsParams) error {
	_, err := q.exec(ctx, q.updateGatewayDynamicFieldsStmt, updateGatewayDynamicFields, arg.RequiresPrivilegedAccess, arg.Name)
	return err
}
