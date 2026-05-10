package sqlite

import (
	"encoding/json"
	"strconv"

	"github.com/huandu/go-sqlbuilder"
	"github.com/sagernet/sing-box/common/byteformats"
)

type Filter func(sb *sqlbuilder.SelectBuilder, value []string) error

func EqualFilter(field string) Filter {
	return func(sb *sqlbuilder.SelectBuilder, value []string) error {
		sb.Where(sb.Equal(field, value[0]))
		return nil
	}
}

func EqualOrNullFilter(field string) Filter {
	return func(sb *sqlbuilder.SelectBuilder, value []string) error {
		sb.Where(sb.Or(sb.Equal(field, value[0]), sb.IsNull(field)))
		return nil
	}
}

func GreaterThanFilter(field string) Filter {
	return func(sb *sqlbuilder.SelectBuilder, value []string) error {
		sb.Where(sb.GreaterThan(field, value[0]))
		return nil
	}
}

func LessThanFilter(field string) Filter {
	return func(sb *sqlbuilder.SelectBuilder, value []string) error {
		sb.Where(sb.LessThan(field, value[0]))
		return nil
	}
}

func GreaterEqualThanFilter(field string) Filter {
	return func(sb *sqlbuilder.SelectBuilder, value []string) error {
		sb.Where(sb.GreaterEqualThan(field, value[0]))
		return nil
	}
}

func LessEqualThanFilter(field string) Filter {
	return func(sb *sqlbuilder.SelectBuilder, value []string) error {
		sb.Where(sb.LessEqualThan(field, value[0]))
		return nil
	}
}

func SpeedGreaterEqualThanFilter(field string) Filter {
	return func(sb *sqlbuilder.SelectBuilder, value []string) error {
		bytesSpeed, err := json.Marshal(value[0])
		if err != nil {
			return err
		}
		speed := &byteformats.NetworkBytesCompat{}
		err = speed.UnmarshalJSON(bytesSpeed)
		if err != nil {
			return err
		}
		sb.Where(sb.GreaterEqualThan(field, speed.Value()))
		return nil
	}
}

func SpeedLessEqualThanFilter(field string) Filter {
	return func(sb *sqlbuilder.SelectBuilder, value []string) error {
		bytesSpeed, err := json.Marshal(value[0])
		if err != nil {
			return err
		}
		speed := &byteformats.NetworkBytesCompat{}
		err = speed.UnmarshalJSON(bytesSpeed)
		if err != nil {
			return err
		}
		sb.Where(sb.LessEqualThan(field, speed.Value()))
		return nil
	}
}

func ExistsAndWhereInFilter(subquery *sqlbuilder.SelectBuilder, field string) Filter {
	return func(sb *sqlbuilder.SelectBuilder, value []string) error {
		values := make([]interface{}, len(value))
		for i, v := range value {
			values[i] = v
		}
		subquery.Where(subquery.In(field, values...))
		sb.Where(sb.Exists(subquery))
		return nil
	}
}

func InFilter(field string) Filter {
	return func(sb *sqlbuilder.SelectBuilder, value []string) error {
		if len(value) == 0 {
			return nil
		}
		values := make([]interface{}, len(value))
		for i, v := range value {
			values[i] = v
		}
		sb.Where(sb.In(field, values...))
		return nil
	}
}

func SortAscFilter() Filter {
	return func(sb *sqlbuilder.SelectBuilder, value []string) error {
		sb.OrderByAsc(value[0])
		return nil
	}
}

func SortDescFilter() Filter {
	return func(sb *sqlbuilder.SelectBuilder, value []string) error {
		sb.OrderByDesc(value[0])
		return nil
	}
}

func ReplacedSortAscFilter(replace map[string]string) Filter {
	return func(sb *sqlbuilder.SelectBuilder, value []string) error {
		if replacedValue, ok := replace[value[0]]; ok {
			sb.OrderByAsc(replacedValue)
		} else {
			sb.OrderByAsc(value[0])
		}
		return nil
	}
}

func ReplacedSortDescFilter(replace map[string]string) Filter {
	return func(sb *sqlbuilder.SelectBuilder, value []string) error {
		if replacedValue, ok := replace[value[0]]; ok {
			sb.OrderByDesc(replacedValue)
		} else {
			sb.OrderByDesc(value[0])
		}
		return nil
	}
}

func LimitFilter() Filter {
	return func(sb *sqlbuilder.SelectBuilder, value []string) error {
		limit, err := strconv.Atoi(value[0])
		if err != nil {
			return err
		}
		sb.Limit(limit)
		return nil
	}
}

func OffsetFilter() Filter {
	return func(sb *sqlbuilder.SelectBuilder, value []string) error {
		offset, err := strconv.Atoi(value[0])
		if err != nil {
			return err
		}
		sb.Offset(offset)
		return nil
	}
}
