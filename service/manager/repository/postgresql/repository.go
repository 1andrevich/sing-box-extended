package postgresql

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/huandu/go-sqlbuilder"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sagernet/sing-box/service/manager/constant"
	"github.com/sagernet/sing/common/byteformats"
)

var (
	squadFilters, nodeFilters, userFilters, bandwidthLimiterFilters, connectionLimiterFilters map[string]Filter
)

type PostgreSQLRepository struct {
	db  *pgxpool.Pool
	ctx context.Context
}

func NewPostgreSQLRepository(ctx context.Context, dsn string) (*PostgreSQLRepository, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	if err := Migrate(db); err != nil && err != migrate.ErrNoChange {
		return nil, err
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	return &PostgreSQLRepository{db: pool, ctx: ctx}, nil
}

func (r *PostgreSQLRepository) CreateSquad(squad constant.SquadCreate) (constant.Squad, error) {
	var s constant.Squad
	now := time.Now()
	err := r.db.QueryRow(r.ctx, `
		INSERT INTO squads
		(
			name,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3)
		RETURNING
			id,
			name,
			created_at,
			updated_at
	`,
		squad.Name,
		now,
		now,
	).Scan(
		&s.ID,
		&s.Name,
		&s.CreatedAt,
		&s.UpdatedAt,
	)
	return s, err
}

func (r *PostgreSQLRepository) GetSquad(id int) (constant.Squad, error) {
	var s constant.Squad
	err := r.db.QueryRow(r.ctx, `
		SELECT
			id,
			name,
			created_at,
			updated_at
		FROM squads
		WHERE id=$1
	`, id).Scan(
		&s.ID,
		&s.Name,
		&s.CreatedAt,
		&s.UpdatedAt,
	)
	return s, err
}

func (r *PostgreSQLRepository) GetSquads(filters map[string][]string) ([]constant.Squad, error) {
	sb := sqlbuilder.PostgreSQL.NewSelectBuilder().
		Select(
			"id",
			"name",
			"created_at",
			"updated_at",
		).
		From("squads")
	for k, v := range filters {
		if f, ok := squadFilters[k]; ok {
			if err := f(sb, v); err != nil {
				return nil, err
			}
		}
	}
	sql, args := sb.Build()
	rows, err := r.db.Query(r.ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []constant.Squad
	for rows.Next() {
		var squad constant.Squad
		if err := rows.Scan(
			&squad.ID,
			&squad.Name,
			&squad.CreatedAt,
			&squad.UpdatedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, squad)
	}
	return result, rows.Err()
}

func (r *PostgreSQLRepository) GetSquadsCount(filters map[string][]string) (int, error) {
	sb := sqlbuilder.PostgreSQL.NewSelectBuilder().
		Select("COUNT(*)").
		From("squads")
	for k, v := range filters {
		if f, ok := squadFilters[k]; ok {
			if err := f(sb, v); err != nil {
				return 0, err
			}
		}
	}
	sql, args := sb.Build()
	var count int
	err := r.db.QueryRow(r.ctx, sql, args...).Scan(&count)
	return count, err
}

func (r *PostgreSQLRepository) UpdateSquad(id int, squad constant.SquadUpdate) (constant.Squad, error) {
	var s constant.Squad
	err := r.db.QueryRow(r.ctx, `
		UPDATE squads
		SET
			name=$1,
			updated_at=$2
		WHERE id=$3
		RETURNING
			id,
			name,
			created_at,
			updated_at
	`,
		squad.Name,
		time.Now(),
		id,
	).Scan(
		&s.ID,
		&s.Name,
		&s.CreatedAt,
		&s.UpdatedAt,
	)
	return s, err
}

func (r *PostgreSQLRepository) DeleteSquad(id int) (constant.Squad, error) {
	var s constant.Squad
	err := r.db.QueryRow(r.ctx, `
		DELETE FROM squads
		WHERE id=$1
		RETURNING
			id,
			name,
			created_at,
			updated_at
	`, id).Scan(
		&s.ID,
		&s.Name,
		&s.CreatedAt,
		&s.UpdatedAt,
	)
	return s, err
}

func (r *PostgreSQLRepository) CreateNode(node constant.NodeCreate) (constant.Node, error) {
	var n constant.Node
	tx, err := r.db.Begin(r.ctx)
	if err != nil {
		return n, err
	}
	defer tx.Rollback(r.ctx)
	now := time.Now()
	err = tx.QueryRow(r.ctx, `
		INSERT INTO nodes (
			uuid,
			name,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3, $4)
		RETURNING 
			uuid,
			name,
			created_at,
			updated_at
	`,
		node.UUID,
		node.Name,
		now,
		now,
	).Scan(
		&n.UUID,
		&n.Name,
		&n.CreatedAt,
		&n.UpdatedAt,
	)
	if err != nil {
		return n, err
	}
	rows := make([][]any, len(node.SquadIDs))
	for i, squadID := range node.SquadIDs {
		rows[i] = []any{node.UUID, squadID}
	}
	_, err = tx.CopyFrom(
		r.ctx,
		pgx.Identifier{"node_to_squad"},
		[]string{"node_uuid", "squad_id"},
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		return n, err
	}
	err = tx.Commit(r.ctx)
	if err != nil {
		return n, err
	}
	return n, err
}

func (r *PostgreSQLRepository) GetNodes(filters map[string][]string) ([]constant.Node, error) {
	sb := sqlbuilder.PostgreSQL.NewSelectBuilder().
		Select(
			"uuid",
			"name",
			`ARRAY(
				SELECT squad_id
				FROM node_to_squad
				WHERE node_to_squad.node_uuid = nodes.uuid
			) as squad_ids`,
			"created_at",
			"updated_at",
		).
		From("nodes")
	for key, value := range filters {
		if filter, ok := nodeFilters[key]; ok {
			if err := filter(sb, value); err != nil {
				return nil, err
			}
		}
	}
	sql, args := sb.Build()
	rows, err := r.db.Query(r.ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []constant.Node
	for rows.Next() {
		var n constant.Node
		if err := rows.Scan(
			&n.UUID,
			&n.Name,
			&n.SquadIDs,
			&n.CreatedAt,
			&n.UpdatedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, n)
	}
	return result, rows.Err()
}

func (r *PostgreSQLRepository) GetNodesCount(filters map[string][]string) (int, error) {
	sb := sqlbuilder.PostgreSQL.NewSelectBuilder().
		Select("COUNT(*)").
		From("nodes")
	for key, value := range filters {
		if filter, ok := nodeFilters[key]; ok {
			if err := filter(sb, value); err != nil {
				return 0, err
			}
		}
	}
	sql, args := sb.Build()
	var count int
	err := r.db.QueryRow(r.ctx, sql, args...).Scan(&count)
	return count, err
}

func (r *PostgreSQLRepository) GetNode(uuid string) (constant.Node, error) {
	var n constant.Node
	err := r.db.QueryRow(r.ctx, `
		SELECT 
			uuid,
			name, 
			ARRAY(
				SELECT squad_id
				FROM node_to_squad
				WHERE node_to_squad.node_uuid = nodes.uuid
			) as squad_ids,
			created_at,
			updated_at
		FROM nodes
		WHERE uuid = $1
	`, uuid).Scan(
		&n.UUID,
		&n.Name,
		&n.SquadIDs,
		&n.CreatedAt,
		&n.UpdatedAt,
	)
	if err != nil && err.Error() == "no rows in result set" {
		return n, constant.ErrNotFound
	}
	return n, err
}

func (r *PostgreSQLRepository) UpdateNode(uuid string, node constant.NodeUpdate) (constant.Node, error) {
	var n constant.Node
	err := r.db.QueryRow(r.ctx, `
		UPDATE nodes
		SET 
			name = $1,
			updated_at = $2
		WHERE uuid = $3
		RETURNING
			uuid,
			name,
			created_at,
			updated_at
	`,
		node.Name,
		time.Now(),
		uuid,
	).Scan(
		&n.UUID,
		&n.Name,
		&n.CreatedAt,
		&n.UpdatedAt,
	)
	return n, err
}

func (r *PostgreSQLRepository) DeleteNode(uuid string) (constant.Node, error) {
	var n constant.Node
	err := r.db.QueryRow(r.ctx, `
		DELETE FROM nodes
		WHERE uuid = $1
		RETURNING 
			uuid,
			name,
			created_at,
			updated_at
	`, uuid).Scan(
		&n.UUID,
		&n.Name,
		&n.CreatedAt,
		&n.UpdatedAt,
	)
	return n, err
}

func (r *PostgreSQLRepository) CreateUser(user constant.UserCreate) (constant.User, error) {
	var u constant.User
	tx, err := r.db.Begin(r.ctx)
	if err != nil {
		return u, err
	}
	defer tx.Rollback(r.ctx)
	now := time.Now()
	err = tx.QueryRow(r.ctx, `
		INSERT INTO users (
			username,
			type,
			inbound,
			uuid,
			password,
			flow,
			alter_id,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING 
			id,
			username,
			type,
			inbound,
			uuid,
			password,
			flow,
			alter_id,
			created_at,
			updated_at
	`,
		user.Username,
		user.Type,
		user.Inbound,
		user.UUID,
		user.Password,
		user.Flow,
		user.AlterID,
		now,
		now,
	).Scan(
		&u.ID,
		&u.Username,
		&u.Type,
		&u.Inbound,
		&u.UUID,
		&u.Password,
		&u.Flow,
		&u.AlterID,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	rows := make([][]any, len(user.SquadIDs))
	for i, squadID := range user.SquadIDs {
		rows[i] = []any{u.ID, squadID}
	}
	_, err = tx.CopyFrom(
		r.ctx,
		pgx.Identifier{"user_to_squad"},
		[]string{"user_id", "squad_id"},
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		return u, err
	}
	u.SquadIDs = user.SquadIDs
	err = tx.Commit(r.ctx)
	if err != nil {
		return u, err
	}
	return u, err
}

func (r *PostgreSQLRepository) GetUsers(filters map[string][]string) ([]constant.User, error) {
	sb := sqlbuilder.PostgreSQL.NewSelectBuilder().
		Select(
			"id",
			`ARRAY(
				SELECT squad_id
				FROM user_to_squad
				WHERE user_to_squad.user_id = users.id
			) as squad_ids`,
			"username",
			"type",
			"inbound",
			"uuid",
			"password",
			"flow",
			"alter_id",
			"created_at",
			"updated_at",
		).
		From("users")
	for key, value := range filters {
		if filter, ok := userFilters[key]; ok {
			if err := filter(sb, value); err != nil {
				return nil, err
			}
		}
	}
	sql, args := sb.Build()
	rows, err := r.db.Query(r.ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []constant.User
	for rows.Next() {
		var u constant.User
		if err := rows.Scan(
			&u.ID,
			&u.SquadIDs,
			&u.Username,
			&u.Type,
			&u.Inbound,
			&u.UUID,
			&u.Password,
			&u.Flow,
			&u.AlterID,
			&u.CreatedAt,
			&u.UpdatedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, u)
	}
	return result, rows.Err()
}

func (r *PostgreSQLRepository) GetUsersCount(filters map[string][]string) (int, error) {
	sb := sqlbuilder.PostgreSQL.NewSelectBuilder().
		Select("COUNT(*)").
		From("users")
	for key, value := range filters {
		if filter, ok := userFilters[key]; ok {
			if err := filter(sb, value); err != nil {
				return 0, err
			}
		}
	}
	sql, args := sb.Build()
	var count int
	err := r.db.QueryRow(r.ctx, sql, args...).Scan(&count)
	return count, err
}

func (r *PostgreSQLRepository) GetUser(id int) (constant.User, error) {
	var u constant.User
	err := r.db.QueryRow(r.ctx, `
		SELECT
			id,
			ARRAY(
				SELECT squad_id
				FROM user_to_squad
				WHERE user_to_squad.user_id = users.id
			) as squad_ids,
			username,
			type,
			inbound,
			uuid,
			password,
			flow,
			alter_id,
			created_at,
			updated_at
		FROM users
		WHERE id = $1
	`, id).Scan(
		&u.ID,
		&u.SquadIDs,
		&u.Username,
		&u.Type,
		&u.Inbound,
		&u.UUID,
		&u.Password,
		&u.Flow,
		&u.AlterID,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	return u, err
}

func (r *PostgreSQLRepository) UpdateUser(id int, user constant.UserUpdate) (constant.User, error) {
	var u constant.User
	err := r.db.QueryRow(r.ctx, `
		UPDATE users
		SET
			uuid = $1,
			password = $2,
			flow = $3,
			alter_id = $4,
			updated_at = $5
		WHERE id = $6
		RETURNING
			id,
			ARRAY(
				SELECT squad_id
				FROM user_to_squad
				WHERE user_to_squad.user_id = users.id
			) as squad_ids,
			username,
			type,
			inbound,
			uuid,
			password,
			flow,
			alter_id,
			created_at,
			updated_at
	`,
		user.UUID,
		user.Password,
		user.Flow,
		user.AlterID,
		time.Now(),
		id,
	).Scan(
		&u.ID,
		&u.SquadIDs,
		&u.Username,
		&u.Type,
		&u.Inbound,
		&u.UUID,
		&u.Password,
		&u.Flow,
		&u.AlterID,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	return u, err
}

func (r *PostgreSQLRepository) DeleteUser(id int) (constant.User, error) {
	var u constant.User
	err := r.db.QueryRow(r.ctx, `
		DELETE FROM users
		WHERE id = $1
		RETURNING
			id,
			ARRAY(
				SELECT squad_id
				FROM user_to_squad
				WHERE user_to_squad.user_id = users.id
			) as squad_ids,
			username,
			type,
			inbound,
			uuid,
			password,
			flow,
			alter_id,
			created_at,
			updated_at
	`, id).Scan(
		&u.ID,
		&u.SquadIDs,
		&u.Username,
		&u.Type,
		&u.Inbound,
		&u.UUID,
		&u.Password,
		&u.Flow,
		&u.AlterID,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	return u, err
}

func (r *PostgreSQLRepository) CreateConnectionLimiter(limiter constant.ConnectionLimiterCreate) (constant.ConnectionLimiter, error) {
	var cl constant.ConnectionLimiter
	tx, err := r.db.Begin(r.ctx)
	if err != nil {
		return cl, err
	}
	defer tx.Rollback(r.ctx)
	now := time.Now()
	err = tx.QueryRow(r.ctx, `
		INSERT INTO connection_limiters
		(
			username,
			outbound,
			strategy,
			connection_type,
			lock_type,
			count,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING
			id,
			username,
			outbound,
			strategy,
			connection_type,
			lock_type,
			count,
			created_at,
			updated_at
	`,
		limiter.Username,
		limiter.Outbound,
		limiter.Strategy,
		limiter.ConnectionType,
		limiter.LockType,
		limiter.Count,
		now,
		now,
	).Scan(
		&cl.ID,
		&cl.Username,
		&cl.Outbound,
		&cl.Strategy,
		&cl.ConnectionType,
		&cl.LockType,
		&cl.Count,
		&cl.CreatedAt,
		&cl.UpdatedAt,
	)
	if err != nil {
		return cl, err
	}
	rows := make([][]any, len(limiter.SquadIDs))
	for i, squadID := range limiter.SquadIDs {
		rows[i] = []any{cl.ID, squadID}
	}
	_, err = tx.CopyFrom(
		r.ctx,
		pgx.Identifier{"connection_limiter_to_squad"},
		[]string{"connection_limiter_id", "squad_id"},
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		return cl, err
	}
	cl.SquadIDs = limiter.SquadIDs
	err = tx.Commit(r.ctx)
	if err != nil {
		return cl, err
	}
	return cl, err
}

func (r *PostgreSQLRepository) GetConnectionLimiter(id int) (constant.ConnectionLimiter, error) {
	var cl constant.ConnectionLimiter
	err := r.db.QueryRow(r.ctx, `
		SELECT
			id,
			ARRAY(
				SELECT squad_id
				FROM connection_limiter_to_squad
				WHERE connection_limiter_to_squad.connection_limiter_id = connection_limiters.id
			) as squad_ids,
			username,
			outbound,
			strategy,
			connection_type,
			lock_type,
			count,
			created_at,
			updated_at
		FROM connection_limiters
		WHERE id=$1
	`, id).Scan(
		&cl.ID,
		&cl.SquadIDs,
		&cl.Username,
		&cl.Outbound,
		&cl.Strategy,
		&cl.ConnectionType,
		&cl.LockType,
		&cl.Count,
		&cl.CreatedAt,
		&cl.UpdatedAt,
	)
	return cl, err
}

func (r *PostgreSQLRepository) GetConnectionLimiters(filters map[string][]string) ([]constant.ConnectionLimiter, error) {
	sb := sqlbuilder.PostgreSQL.NewSelectBuilder().
		Select(
			"id",
			`ARRAY(
				SELECT squad_id
				FROM connection_limiter_to_squad
				WHERE connection_limiter_to_squad.connection_limiter_id = connection_limiters.id
			) as squad_ids`,
			"username",
			"outbound",
			"strategy",
			"connection_type",
			"lock_type",
			"count",
			"created_at",
			"updated_at",
		).
		From("connection_limiters")
	for k, v := range filters {
		if f, ok := connectionLimiterFilters[k]; ok {
			if err := f(sb, v); err != nil {
				return nil, err
			}
		}
	}
	sql, args := sb.Build()
	rows, err := r.db.Query(r.ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []constant.ConnectionLimiter
	for rows.Next() {
		var cl constant.ConnectionLimiter
		if err := rows.Scan(
			&cl.ID,
			&cl.SquadIDs,
			&cl.Username,
			&cl.Outbound,
			&cl.Strategy,
			&cl.ConnectionType,
			&cl.LockType,
			&cl.Count,
			&cl.CreatedAt,
			&cl.UpdatedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, cl)
	}

	return result, rows.Err()
}

func (r *PostgreSQLRepository) GetConnectionLimitersCount(filters map[string][]string) (int, error) {
	sb := sqlbuilder.PostgreSQL.NewSelectBuilder().
		Select("COUNT(*)").
		From("connection_limiters")

	for k, v := range filters {
		if f, ok := connectionLimiterFilters[k]; ok {
			if err := f(sb, v); err != nil {
				return 0, err
			}
		}
	}
	sql, args := sb.Build()
	var count int
	err := r.db.QueryRow(r.ctx, sql, args...).Scan(&count)
	return count, err
}

func (r *PostgreSQLRepository) UpdateConnectionLimiter(id int, limiter constant.ConnectionLimiterUpdate) (constant.ConnectionLimiter, error) {
	var cl constant.ConnectionLimiter
	err := r.db.QueryRow(r.ctx, `
		UPDATE connection_limiters
		SET
			strategy=$1,
			connection_type=$2,
			lock_type=$3,
			count=$4,
			updated_at=$5
		WHERE id=$6
		RETURNING
			id,
			ARRAY(
				SELECT squad_id
				FROM connection_limiter_to_squad
				WHERE connection_limiter_to_squad.connection_limiter_id = connection_limiters.id
			) as squad_ids,
			username,
			outbound,
			strategy,
			connection_type,
			lock_type,
			count,
			created_at,
			updated_at
	`,
		limiter.Strategy,
		limiter.ConnectionType,
		limiter.LockType,
		limiter.Count,
		time.Now(),
		id,
	).Scan(
		&cl.ID,
		&cl.SquadIDs,
		&cl.Username,
		&cl.Outbound,
		&cl.Strategy,
		&cl.ConnectionType,
		&cl.LockType,
		&cl.Count,
		&cl.CreatedAt,
		&cl.UpdatedAt,
	)
	return cl, err
}

func (r *PostgreSQLRepository) DeleteConnectionLimiter(id int) (constant.ConnectionLimiter, error) {
	var cl constant.ConnectionLimiter
	err := r.db.QueryRow(r.ctx, `
		DELETE FROM connection_limiters
		WHERE id=$1
		RETURNING
			id,
			ARRAY(
				SELECT squad_id
				FROM connection_limiter_to_squad
				WHERE connection_limiter_to_squad.connection_limiter_id = connection_limiters.id
			) as squad_ids,
			username,
			outbound,
			strategy,
			connection_type,
			lock_type,
			count,
			created_at,
			updated_at
	`, id).Scan(
		&cl.ID,
		&cl.SquadIDs,
		&cl.Username,
		&cl.Outbound,
		&cl.Strategy,
		&cl.ConnectionType,
		&cl.LockType,
		&cl.Count,
		&cl.CreatedAt,
		&cl.UpdatedAt,
	)
	return cl, err
}

func (r *PostgreSQLRepository) CreateBandwidthLimiter(limiter constant.BandwidthLimiterCreate) (constant.BandwidthLimiter, error) {
	var bl constant.BandwidthLimiter
	tx, err := r.db.Begin(r.ctx)
	if err != nil {
		return bl, err
	}
	defer tx.Rollback(r.ctx)
	bytesSpeed, err := json.Marshal(limiter.Speed)
	if err != nil {
		return bl, err
	}
	raw := &byteformats.NetworkBytesCompat{}
	if err = raw.UnmarshalJSON(bytesSpeed); err != nil {
		return bl, err
	}
	now := time.Now()
	err = tx.QueryRow(r.ctx, `
		INSERT INTO bandwidth_limiters
		(
			username,
			outbound,
			strategy,
			mode,
			connection_type,
			speed,
			raw_speed,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING
			id,
			username,
			outbound,
			strategy,
			mode,
			connection_type,
			speed,
			raw_speed,
			created_at,
			updated_at
	`,
		limiter.Username,
		limiter.Outbound,
		limiter.Strategy,
		limiter.Mode,
		limiter.ConnectionType,
		limiter.Speed,
		raw.Value(),
		now,
		now,
	).Scan(
		&bl.ID,
		&bl.Username,
		&bl.Outbound,
		&bl.Strategy,
		&bl.Mode,
		&bl.ConnectionType,
		&bl.Speed,
		&bl.RawSpeed,
		&bl.CreatedAt,
		&bl.UpdatedAt,
	)
	if err != nil {
		return bl, err
	}
	rows := make([][]any, len(limiter.SquadIDs))
	for i, squadID := range limiter.SquadIDs {
		rows[i] = []any{bl.ID, squadID}
	}
	_, err = tx.CopyFrom(
		r.ctx,
		pgx.Identifier{"bandwidth_limiter_to_squad"},
		[]string{"bandwidth_limiter_id", "squad_id"},
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		return bl, err
	}
	bl.SquadIDs = limiter.SquadIDs
	err = tx.Commit(r.ctx)
	if err != nil {
		return bl, err
	}
	return bl, err
}

func (r *PostgreSQLRepository) GetBandwidthLimiter(id int) (constant.BandwidthLimiter, error) {
	var bl constant.BandwidthLimiter
	err := r.db.QueryRow(r.ctx, `
		SELECT
			id,
			ARRAY(
				SELECT squad_id
				FROM bandwidth_limiter_to_squad
				WHERE bandwidth_limiter_to_squad.bandwidth_limiter_id = bandwidth_limiters.id
			) as squad_ids,
			username,
			outbound,
			strategy,
			mode,
			connection_type,
			speed,
			raw_speed,
			created_at,
			updated_at
		FROM bandwidth_limiters
		WHERE id=$1
	`, id).Scan(
		&bl.ID,
		&bl.SquadIDs,
		&bl.Username,
		&bl.Outbound,
		&bl.Strategy,
		&bl.Mode,
		&bl.ConnectionType,
		&bl.Speed,
		&bl.RawSpeed,
		&bl.CreatedAt,
		&bl.UpdatedAt,
	)
	return bl, err
}

func (r *PostgreSQLRepository) GetBandwidthLimiters(filters map[string][]string) ([]constant.BandwidthLimiter, error) {
	sb := sqlbuilder.PostgreSQL.NewSelectBuilder().
		Select(
			"id",
			`ARRAY(
				SELECT squad_id
				FROM bandwidth_limiter_to_squad
				WHERE bandwidth_limiter_to_squad.bandwidth_limiter_id = bandwidth_limiters.id
			) as squad_ids`,
			"username",
			"outbound",
			"strategy",
			"mode",
			"connection_type",
			"speed",
			"raw_speed",
			"created_at",
			"updated_at",
		).
		From("bandwidth_limiters")

	for k, v := range filters {
		if f, ok := bandwidthLimiterFilters[k]; ok {
			if err := f(sb, v); err != nil {
				return nil, err
			}
		}
	}
	sql, args := sb.Build()
	rows, err := r.db.Query(r.ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []constant.BandwidthLimiter
	for rows.Next() {
		var bl constant.BandwidthLimiter
		if err := rows.Scan(
			&bl.ID,
			&bl.SquadIDs,
			&bl.Username,
			&bl.Outbound,
			&bl.Strategy,
			&bl.Mode,
			&bl.ConnectionType,
			&bl.Speed,
			&bl.RawSpeed,
			&bl.CreatedAt,
			&bl.UpdatedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, bl)
	}
	return result, rows.Err()
}

func (r *PostgreSQLRepository) GetBandwidthLimitersCount(filters map[string][]string) (int, error) {
	sb := sqlbuilder.PostgreSQL.NewSelectBuilder().
		Select("COUNT(*)").
		From("bandwidth_limiters")
	for k, v := range filters {
		if f, ok := bandwidthLimiterFilters[k]; ok {
			if err := f(sb, v); err != nil {
				return 0, err
			}
		}
	}
	sql, args := sb.Build()
	var count int
	err := r.db.QueryRow(r.ctx, sql, args...).Scan(&count)
	return count, err
}

func (r *PostgreSQLRepository) UpdateBandwidthLimiter(id int, limiter constant.BandwidthLimiterUpdate) (constant.BandwidthLimiter, error) {
	var bl constant.BandwidthLimiter
	bytesSpeed, err := json.Marshal(limiter.Speed)
	if err != nil {
		return bl, err
	}
	raw := &byteformats.NetworkBytesCompat{}
	if err = raw.UnmarshalJSON(bytesSpeed); err != nil {
		return bl, err
	}
	err = r.db.QueryRow(r.ctx, `
		UPDATE bandwidth_limiters
		SET
			username=$1,
			outbound=$2,
			strategy=$3,
			mode=$4,
			connection_type=$5,
			speed=$6,
			raw_speed=$7,
			updated_at=$8
		WHERE id=$9
		RETURNING
			id,
			ARRAY(
				SELECT squad_id
				FROM bandwidth_limiter_to_squad
				WHERE bandwidth_limiter_to_squad.bandwidth_limiter_id = bandwidth_limiters.id
			) as squad_ids,
			username,
			outbound,
			strategy,
			mode,
			connection_type,
			speed,
			raw_speed,
			created_at,
			updated_at
	`,
		limiter.Username,
		limiter.Outbound,
		limiter.Strategy,
		limiter.Mode,
		limiter.ConnectionType,
		limiter.Speed,
		raw.Value(),
		time.Now(),
		id,
	).Scan(
		&bl.ID,
		&bl.SquadIDs,
		&bl.Username,
		&bl.Outbound,
		&bl.Strategy,
		&bl.Mode,
		&bl.ConnectionType,
		&bl.Speed,
		&bl.RawSpeed,
		&bl.CreatedAt,
		&bl.UpdatedAt,
	)
	return bl, err
}

func (r *PostgreSQLRepository) DeleteBandwidthLimiter(id int) (constant.BandwidthLimiter, error) {
	var bl constant.BandwidthLimiter
	err := r.db.QueryRow(r.ctx, `
		DELETE FROM bandwidth_limiters
		WHERE id=$1
		RETURNING
			id,
			ARRAY(
				SELECT squad_id
				FROM bandwidth_limiter_to_squad
				WHERE bandwidth_limiter_to_squad.bandwidth_limiter_id = bandwidth_limiters.id
			) as squad_ids,
			username,
			outbound,
			strategy,
			mode,
			connection_type,
			speed,
			raw_speed,
			created_at,
			updated_at
	`, id).Scan(
		&bl.ID,
		&bl.SquadIDs,
		&bl.Username,
		&bl.Outbound,
		&bl.Strategy,
		&bl.Mode,
		&bl.ConnectionType,
		&bl.Speed,
		&bl.RawSpeed,
		&bl.CreatedAt,
		&bl.UpdatedAt,
	)
	return bl, err
}

func init() {
	squadFilters = map[string]Filter{
		"id":               EqualFilter("id"),
		"pk":               EqualFilter("id"),
		"name":             EqualFilter("name"),
		"created_at_start": GreaterThanFilter("created_at"),
		"created_at_end":   LessThanFilter("created_at"),
		"updated_at_start": GreaterThanFilter("updated_at"),
		"updated_at_end":   LessThanFilter("updated_at"),
		"sort_asc":         SortAscFilter(),
		"sort_desc":        SortDescFilter(),
		"offset":           OffsetFilter(),
		"limit":            LimitFilter(),
	}
	nodeFilters = map[string]Filter{
		"uuid": EqualFilter("uuid"),
		"pk":   EqualFilter("uuid"),
		"name": EqualFilter("name"),
		"squad_id_in": ExistsAndWhereInFilter(
			sqlbuilder.PostgreSQL.NewSelectBuilder().
				Select(
					"squad_id",
				).
				Where(
					"node_to_squad.node_uuid = nodes.uuid",
				).
				From(
					"node_to_squad",
				),
			"node_to_squad.squad_id",
		),
		"created_at_start": GreaterThanFilter("created_at"),
		"created_at_end":   LessThanFilter("created_at"),
		"updated_at_start": GreaterThanFilter("updated_at"),
		"updated_at_end":   LessThanFilter("updated_at"),
		"sort_asc":         SortAscFilter(),
		"sort_desc":        SortDescFilter(),
		"offset":           OffsetFilter(),
		"limit":            LimitFilter(),
	}
	userFilters = map[string]Filter{
		"id": EqualFilter("id"),
		"pk": EqualFilter("id"),
		"squad_id_in": ExistsAndWhereInFilter(
			sqlbuilder.PostgreSQL.NewSelectBuilder().
				Select(
					"squad_id",
				).
				Where(
					"user_to_squad.user_id = users.id",
				).
				From(
					"user_to_squad",
				),
			"user_to_squad.squad_id",
		),
		"username":         EqualFilter("username"),
		"type":             EqualFilter("type"),
		"inbound":          EqualFilter("inbound"),
		"created_at_start": GreaterThanFilter("created_at"),
		"created_at_end":   LessThanFilter("created_at"),
		"updated_at_start": GreaterThanFilter("updated_at"),
		"updated_at_end":   LessThanFilter("updated_at"),
		"sort_asc":         SortAscFilter(),
		"sort_desc":        SortDescFilter(),
		"offset":           OffsetFilter(),
		"limit":            LimitFilter(),
	}
	connectionLimiterFilters = map[string]Filter{
		"id": EqualFilter("id"),
		"pk": EqualFilter("id"),
		"squad_id_in": ExistsAndWhereInFilter(
			sqlbuilder.PostgreSQL.NewSelectBuilder().
				Select(
					"squad_id",
				).
				Where(
					"connection_limiter_to_squad.connection_limiter_id = connection_limiters.id",
				).
				From(
					"connection_limiter_to_squad",
				),
			"connection_limiter_to_squad.squad_id",
		),
		"strategy":         EqualFilter("strategy"),
		"username":         EqualFilter("username"),
		"outbound":         EqualFilter("outbound"),
		"connection_type":  EqualFilter("connection_type"),
		"lock_type":        EqualFilter("lock_type"),
		"created_at_start": GreaterThanFilter("created_at"),
		"created_at_end":   LessThanFilter("created_at"),
		"updated_at_start": GreaterThanFilter("updated_at"),
		"updated_at_end":   LessThanFilter("updated_at"),
		"sort_asc":         SortAscFilter(),
		"sort_desc":        SortDescFilter(),
		"offset":           OffsetFilter(),
		"limit":            LimitFilter(),
	}
	bandwidthLimiterFilters = map[string]Filter{
		"id": EqualFilter("id"),
		"pk": EqualFilter("id"),
		"squad_id_in": ExistsAndWhereInFilter(
			sqlbuilder.PostgreSQL.NewSelectBuilder().
				Select(
					"squad_id",
				).
				Where(
					"bandwidth_limiter_to_squad.bandwidth_limiter_id = bandwidth_limiters.id",
				).
				From(
					"bandwidth_limiter_to_squad",
				),
			"bandwidth_limiter_to_squad.squad_id",
		),
		"strategy":         EqualFilter("strategy"),
		"mode":             EqualFilter("mode"),
		"type":             EqualFilter("type"),
		"username":         EqualFilter("username"),
		"down_start":       SpeedGreaterEqualThanFilter("raw_down"),
		"down_end":         SpeedLessEqualThanFilter("raw_down"),
		"up_start":         SpeedGreaterEqualThanFilter("raw_up"),
		"up_end":           SpeedLessEqualThanFilter("raw_up"),
		"created_at_start": GreaterThanFilter("created_at"),
		"created_at_end":   LessThanFilter("created_at"),
		"updated_at_start": GreaterThanFilter("updated_at"),
		"updated_at_end":   LessThanFilter("updated_at"),
		"sort_asc":         ReplacedSortAscFilter(map[string]string{"down": "raw_down", "up": "raw_up"}),
		"sort_desc":        ReplacedSortDescFilter(map[string]string{"down": "raw_down", "up": "raw_up"}),
		"offset":           OffsetFilter(),
		"limit":            LimitFilter(),
	}
}
