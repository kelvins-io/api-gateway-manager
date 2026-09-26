package database

import (
	"fmt"

	"github.com/kelvins-io/api-gateway-manager/internal/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
	}
	return db, nil
}

func AutoMigrate(db *gorm.DB) error {
	if err := renameAccessColumns(db); err != nil {
		return err
	}
	if err := migrateEntityIDsToBigint(db); err != nil {
		return err
	}
	if err := dedupeScopedNames(db); err != nil {
		return err
	}
	if err := db.AutoMigrate(
		&model.User{},
		&model.Space{},
		&model.SpaceMember{},
		&model.Gateway{},
		&model.APIGroup{},
		&model.API{},
		&model.APIVersion{},
		&model.Upstream{},
		&model.UpstreamTarget{},
		&model.UpstreamGateway{},
		&model.Consumer{},
		&model.ConsumerCredential{},
		&model.ConsumerGateway{},
		&model.Plugin{},
	); err != nil {
		return err
	}
	if err := backfillApprovalStatuses(db); err != nil {
		return err
	}
	return ensureJoinPrimaryKeys(db)
}

// dedupeScopedNames renames duplicate names so unique indexes can be created.
// The lowest id keeps the original name; later rows get a "-{id}" suffix.
func dedupeScopedNames(db *gorm.DB) error {
	if db.Migrator().HasTable(&model.API{}) {
		if err := db.Exec(`
			UPDATE apis a
			SET name = left(a.name, 128 - length('-' || a.id::text)) || '-' || a.id::text
			WHERE EXISTS (
				SELECT 1 FROM apis b
				WHERE b.group_id = a.group_id AND b.name = a.name AND b.id < a.id
			)
		`).Error; err != nil {
			return fmt.Errorf("dedupe api names: %w", err)
		}
	}
	if db.Migrator().HasTable(&model.APIGroup{}) {
		if err := db.Exec(`
			UPDATE api_groups a
			SET name = left(a.name, 128 - length('-' || a.id::text)) || '-' || a.id::text
			WHERE EXISTS (
				SELECT 1 FROM api_groups b
				WHERE b.space_id = a.space_id AND b.name = a.name AND b.id < a.id
			)
		`).Error; err != nil {
			return fmt.Errorf("dedupe group names: %w", err)
		}
	}
	return nil
}

func backfillApprovalStatuses(db *gorm.DB) error {
	if db.Migrator().HasTable(&model.Space{}) {
		if err := db.Exec(`UPDATE spaces SET status = ? WHERE status IS NULL OR status = ''`, model.SpaceStatusActive).Error; err != nil {
			return fmt.Errorf("backfill space status: %w", err)
		}
	}
	if db.Migrator().HasTable(&model.SpaceMember{}) {
		if err := db.Exec(`UPDATE space_members SET status = ? WHERE status IS NULL OR status = ''`, model.MemberStatusActive).Error; err != nil {
			return fmt.Errorf("backfill space member status: %w", err)
		}
	}
	return nil
}

func ensureJoinPrimaryKeys(db *gorm.DB) error {
	pairs := []struct{ table, cols string }{
		{"api_plugins", "api_id, plugin_id"},
		{"api_consumers", "api_id, consumer_id"},
	}
	for _, pair := range pairs {
		if !db.Migrator().HasTable(pair.table) {
			continue
		}
		stmt := fmt.Sprintf(`
			DO $$ BEGIN
				IF NOT EXISTS (
					SELECT 1 FROM pg_constraint
					WHERE conname = '%s_pkey' AND conrelid = '%s'::regclass
				) THEN
					ALTER TABLE %s ADD CONSTRAINT %s_pkey PRIMARY KEY (%s);
				END IF;
			END $$`, pair.table, pair.table, pair.table, pair.table, pair.cols)
		if err := db.Exec(stmt).Error; err != nil {
			return fmt.Errorf("join key %s: %w", pair.table, err)
		}
	}
	return nil
}

func renameAccessColumns(db *gorm.DB) error {
	if !db.Migrator().HasTable(&model.API{}) {
		return nil
	}
	pairs := [][2]string{
		{"path", "access_path"},
		{"methods", "access_methods"},
		{"strip_path", "access_strip_path"},
		{"protocol", "service_protocol"},
		{"host_kind", "service_host_kind"},
		{"host", "service_host"},
		{"upstream_id", "service_upstream_id"},
		{"port", "service_port"},
		{"retries", "service_retries"},
		{"connect_timeout", "service_connect_timeout"},
		{"write_timeout", "service_write_timeout"},
		{"read_timeout", "service_read_timeout"},
	}
	for _, pair := range pairs {
		hasOld := db.Migrator().HasColumn(&model.API{}, pair[0])
		hasNew := db.Migrator().HasColumn(&model.API{}, pair[1])
		if hasOld && !hasNew {
			if err := db.Migrator().RenameColumn(&model.API{}, pair[0], pair[1]); err != nil {
				return fmt.Errorf("rename apis.%s: %w", pair[0], err)
			}
		}
	}
	return nil
}

// migrateEntityIDsToBigint converts spaces, api_groups, and apis primary keys
// from uuid back to bigint, and rewrites every foreign key that points at them.
func migrateEntityIDsToBigint(db *gorm.DB) error {
	if !db.Migrator().HasTable("spaces") {
		return nil
	}
	var dataType string
	if err := db.Raw(`
		SELECT data_type FROM information_schema.columns
		WHERE table_schema = CURRENT_SCHEMA() AND table_name = 'spaces' AND column_name = 'id'
	`).Scan(&dataType).Error; err != nil {
		return fmt.Errorf("inspect spaces.id: %w", err)
	}
	if dataType == "" || dataType == "bigint" {
		return nil
	}

	stmts := []string{
		`CREATE SEQUENCE IF NOT EXISTS spaces_id_seq`,
		`ALTER TABLE spaces ADD COLUMN IF NOT EXISTS id_num bigint`,
		`UPDATE spaces SET id_num = nextval('spaces_id_seq') WHERE id_num IS NULL`,
	}
	if db.Migrator().HasTable("api_groups") {
		stmts = append(stmts,
			`CREATE SEQUENCE IF NOT EXISTS api_groups_id_seq`,
			`ALTER TABLE api_groups ADD COLUMN IF NOT EXISTS id_num bigint`,
			`UPDATE api_groups SET id_num = nextval('api_groups_id_seq') WHERE id_num IS NULL`,
			`ALTER TABLE api_groups ADD COLUMN IF NOT EXISTS space_id_num bigint`,
			`UPDATE api_groups g SET space_id_num = s.id_num FROM spaces s WHERE g.space_id = s.id AND g.space_id_num IS NULL`,
		)
	}
	if db.Migrator().HasTable("apis") {
		stmts = append(stmts,
			`CREATE SEQUENCE IF NOT EXISTS apis_id_seq`,
			`ALTER TABLE apis ADD COLUMN IF NOT EXISTS id_num bigint`,
			`UPDATE apis SET id_num = nextval('apis_id_seq') WHERE id_num IS NULL`,
			`ALTER TABLE apis ADD COLUMN IF NOT EXISTS group_id_num bigint`,
			`UPDATE apis a SET group_id_num = g.id_num FROM api_groups g WHERE a.group_id = g.id AND a.group_id_num IS NULL`,
		)
	}
	refs := []struct{ table, column, parent string }{
		{"space_members", "space_id", "spaces"},
		{"plugins", "space_id", "spaces"},
		{"upstreams", "space_id", "spaces"},
		{"consumers", "space_id", "spaces"},
		{"api_versions", "api_id", "apis"},
		{"api_plugins", "api_id", "apis"},
		{"api_consumers", "api_id", "apis"},
	}
	for _, ref := range refs {
		if !db.Migrator().HasTable(ref.table) || !db.Migrator().HasColumn(ref.table, ref.column) {
			continue
		}
		stmts = append(stmts,
			fmt.Sprintf(`ALTER TABLE %s ADD COLUMN IF NOT EXISTS %s_num bigint`, ref.table, ref.column),
			fmt.Sprintf(`UPDATE %s c SET %s_num = p.id_num FROM %s p WHERE c.%s = p.id AND c.%s_num IS NULL`,
				ref.table, ref.column, ref.parent, ref.column, ref.column),
		)
	}

	return db.Transaction(func(tx *gorm.DB) error {
		if err := dropForeignKeys(tx); err != nil {
			return err
		}
		for _, stmt := range stmts {
			if err := tx.Exec(stmt).Error; err != nil {
				return fmt.Errorf("%s: %w", stmt, err)
			}
		}
		swaps := []struct{ table, pk string }{
			{"spaces", "spaces_pkey"},
		}
		if tx.Migrator().HasTable("api_groups") {
			swaps = append(swaps, struct{ table, pk string }{"api_groups", "api_groups_pkey"})
		}
		if tx.Migrator().HasTable("apis") {
			swaps = append(swaps, struct{ table, pk string }{"apis", "apis_pkey"})
		}
		for _, s := range swaps {
			if err := swapIDColumn(tx, s.table, "id", "id_num", true, s.pk); err != nil {
				return err
			}
			if err := attachBigintSequence(tx, s.table); err != nil {
				return err
			}
		}
		plain := []struct{ table, column string }{
			{"api_groups", "space_id"},
			{"apis", "group_id"},
			{"space_members", "space_id"},
			{"plugins", "space_id"},
			{"upstreams", "space_id"},
			{"consumers", "space_id"},
			{"api_versions", "api_id"},
			{"api_plugins", "api_id"},
			{"api_consumers", "api_id"},
		}
		for _, col := range plain {
			if !tx.Migrator().HasTable(col.table) || !tx.Migrator().HasColumn(col.table, col.column+"_num") {
				continue
			}
			if err := swapIDColumn(tx, col.table, col.column, col.column+"_num", false, ""); err != nil {
				return err
			}
		}
		return nil
	})
}

func attachBigintSequence(tx *gorm.DB, table string) error {
	seq := table + "_id_seq"
	stmts := []string{
		fmt.Sprintf(`ALTER TABLE %s ALTER COLUMN id SET DEFAULT nextval('%s')`, table, seq),
		fmt.Sprintf(`ALTER SEQUENCE %s OWNED BY %s.id`, seq, table),
		fmt.Sprintf(`SELECT setval('%s', GREATEST(COALESCE((SELECT MAX(id) FROM %s), 1), 1))`, seq, table),
	}
	for _, stmt := range stmts {
		if err := tx.Exec(stmt).Error; err != nil {
			return fmt.Errorf("%s: %w", stmt, err)
		}
	}
	return nil
}

func dropForeignKeys(tx *gorm.DB) error {
	type fk struct {
		Name  string
		Table string
	}
	var rows []fk
	err := tx.Raw(`
		SELECT c.conname AS name, t.relname AS table
		FROM pg_constraint c
		JOIN pg_class t ON t.oid = c.conrelid
		JOIN pg_namespace n ON n.oid = t.relnamespace
		WHERE c.contype = 'f' AND n.nspname = CURRENT_SCHEMA()
	`).Scan(&rows).Error
	if err != nil {
		return fmt.Errorf("list foreign keys: %w", err)
	}
	for _, row := range rows {
		if err := tx.Exec(fmt.Sprintf(`ALTER TABLE %s DROP CONSTRAINT IF EXISTS %s`, row.Table, row.Name)).Error; err != nil {
			return fmt.Errorf("drop fk %s: %w", row.Name, err)
		}
	}
	return nil
}

func swapIDColumn(tx *gorm.DB, table, oldCol, newCol string, primary bool, pkName string) error {
	if primary && pkName != "" {
		if err := tx.Exec(fmt.Sprintf(`ALTER TABLE %s DROP CONSTRAINT IF EXISTS %s`, table, pkName)).Error; err != nil {
			return err
		}
	}
	if err := tx.Exec(fmt.Sprintf(`ALTER TABLE %s DROP COLUMN %s CASCADE`, table, oldCol)).Error; err != nil {
		return fmt.Errorf("drop %s.%s: %w", table, oldCol, err)
	}
	if err := tx.Exec(fmt.Sprintf(`ALTER TABLE %s RENAME COLUMN %s TO %s`, table, newCol, oldCol)).Error; err != nil {
		return fmt.Errorf("rename %s.%s: %w", table, newCol, err)
	}
	if err := tx.Exec(fmt.Sprintf(`ALTER TABLE %s ALTER COLUMN %s SET NOT NULL`, table, oldCol)).Error; err != nil {
		return err
	}
	if primary {
		if err := tx.Exec(fmt.Sprintf(`ALTER TABLE %s ADD PRIMARY KEY (%s)`, table, oldCol)).Error; err != nil {
			return fmt.Errorf("pk %s: %w", table, err)
		}
	}
	return nil
}
