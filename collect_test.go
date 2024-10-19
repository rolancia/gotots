package gotots_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gotots"
	"strings"
	"testing"
)

func collect(t *testing.T, ty any, ctx *gotots.Context) string {
	if ctx == nil {
		cfg := gotots.Config{IndentWithTabs: true}
		cfg.Init()
		ctx = gotots.NewContext(cfg)
	}
	require.NoError(t, gotots.Collect(ctx, ty))
	out := new(strings.Builder)
	out.WriteByte('\n')
	require.NoError(t, gotots.WriteFromContext(ctx, out))
	return out.String()[:len(out.String())-2] // remove trailing newline
}

type Employee struct {
	ID       int64    `json:"id"`
	Name     string   `json:"name"`
	Position string   `json:"position"`
	Projects []string `json:"projects,omitempty"`
	Address  struct {
		Street      string `json:"street"`
		City        string `json:"city"`
		CountryCode string `json:"country_code"`
		GeoLocation struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		} `json:"geo_location"`
	} `json:"address"`
	Departments []Department `json:"departments"`
	Skills      map[string]struct {
		Level       int  `json:"level"`
		Certified   bool `json:"certified"`
		CertDetails *struct {
			CertName  string `json:"cert_name"`
			IssueDate string `json:"issue_date"`
		} `json:"cert_details,omitempty"`
	} `json:"skills"`
	Metadata map[int][]struct {
		Description string `json:"description"`
		Important   bool   `json:"important"`
	} `json:"metadata"`
	CustomField string `json:"custom" tstype:"CustomType"`
	Data        []byte `json:"data"`
}

type Department struct {
	Name     string    `json:"name"`
	Manager  *Employee `json:"manager,omitempty"`
	SubTeams []struct {
		TeamName         string      `json:"team_name"`
		TeamLead         *Employee   `json:"team_lead,omitempty"`
		Members          []*Employee `json:"members,omitempty"`
		Responsibilities map[string]struct {
			Description string `json:"description"`
			Required    bool   `json:"required"`
		} `json:"responsibilities"`
	} `json:"sub_teams"`
}

func TestGotots(t *testing.T) {
	t.Run("Works", func(t *testing.T) {
		cfg := gotots.Config{IndentWithTabs: true}
		cfg.Init()
		ctx := gotots.NewContext(cfg)
		ctx.AddCustomHeader("import { CustomType } from './custom';")
		assert.Equal(t, `
import { CustomType } from './custom';

export interface Gotots_test_Employee {
	id: bigint;
	name: string;
	position: string;
	projects?: string[];
	address: {
		street: string;
		city: string;
		country_code: string;
		geo_location: {
			latitude: number;
			longitude: number;
		};
	};
	departments?: Gotots_test_Department[];
	skills: { [key: string]: {
		level: number;
		certified: boolean;
		cert_details?: {
			cert_name: string;
			issue_date: string;
		};
	} };
	metadata: { [key: number]: ({
		description: string;
		important: boolean;
	}[] | undefined) };
	custom: CustomType;
	data?: number[];
}

export interface Gotots_test_Department {
	name: string;
	manager?: Gotots_test_Employee;
	sub_teams?: {
		team_name: string;
		team_lead?: Gotots_test_Employee;
		members?: Gotots_test_Employee[];
		responsibilities: { [key: string]: {
			description: string;
			required: boolean;
		} };
	}[];
}`, collect(t, Department{}, ctx))
	})

	t.Run("Types are defined with correct types", func(t *testing.T) {
		type T struct {
			Int     int     `json:"int"`
			Int8    int8    `json:"int8"`
			Int16   int16   `json:"int16"`
			Int32   int32   `json:"int32"`
			Int64   int64   `json:"int64"`
			Uint    uint    `json:"uint"`
			Uint8   uint8   `json:"uint8"`
			Uint16  uint16  `json:"uint16"`
			Uint32  uint32  `json:"uint32"`
			Uint64  uint64  `json:"uint64"`
			Float32 float32 `json:"float32"`
			Float64 float64 `json:"float64"`
			String  string  `json:"string"`
			Bool    bool    `json:"bool"`
		}
		assert.Equal(t, `
export interface Gotots_test_T {
	int: number;
	int8: number;
	int16: number;
	int32: number;
	int64: bigint;
	uint: number;
	uint8: number;
	uint16: number;
	uint32: number;
	uint64: bigint;
	float32: number;
	float64: number;
	string: string;
	bool: boolean;
}`, collect(t, T{}, nil))
	})

	t.Run("Works with multiple types", func(t *testing.T) {
		type T1 struct {
			A int `json:"a"`
		}
		type T2 struct {
			B string `json:"b"`
		}
		cfg := &gotots.Config{IndentWithTabs: true}
		cfg.Init()
		ctx := gotots.NewContext(*cfg)
		require.NoError(t, gotots.Collect(ctx, T1{}))
		require.NoError(t, gotots.Collect(ctx, T2{}))
		out := new(strings.Builder)
		out.WriteByte('\n')
		require.NoError(t, gotots.WriteFromContext(ctx, out))
		assert.Equal(t, `
export interface Gotots_test_T1 {
	a: number;
}

export interface Gotots_test_T2 {
	b: string;
}

`, out.String())
	})
}
