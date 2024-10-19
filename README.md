# GOtoTS
Convert Go struct to TypeScript interfaces.

### Usage
```go
cfg := gotots.Config{IndentWithTabs: true}
cfg.Init()
ctx := gotots.NewContext(cfg)
ctx.AddCustomHeader("import { SomeCustomType } from './some-custom-type';")
_ = gotots.Collect(ctx, YourGoStruct{})
_ = gotots.WriteFromContext(ctx, io.StdOut)
```

### Example

Here is an example of a Go struct and the corresponding TypeScript interface that would be generated:

```go
// go
type Parent struct {
    Name string `json:"name"`
}

type Employee struct {
    ID       int64    `json:"id"`
    Name     string   `json:"name"`
    Position string   `json:"position"`
    Projects []string `json:"projects,omitempty"`
    Information struct {
        Age int `json:"age"`
        Address struct {
            Street string `json:"street"`
            City   string `json:"city"`
        } `json:"address"`
    } `json:"information"`
    // tstype tag can be used to specify the type of the field in the generated ts
    // the type can be imported in generted ts by adding a custom header to the context by calling context.AddCustomHeader
    CustomField string `json:"custom" tstype:"[]SomeCustomType"`
    Parent Parent `json:"parent"`
}
```
```typescript
// generated ts
import { SomeCustomType } from './some-custom-type';

export interface Gotots_test_Parent {
    name: string;
}

export interface Gotots_test_Employee {
    id: bigint;
    name: string;
    position: string;
    projects?: string[];
    information: {
        age: number;
        address: {
            street: string;
            city: string;
        };
    };
    custom: []SomeCustomType;
    parent: Gotots_test_Parent;
}
```

Generate multiple interfaces in a single output:
```go
cfg := gotots.Config{IndentWithTabs: true}
cfg.Init()
ctx := gotots.NewContext(cfg)
_ = gotots.Collect(ctx, T1{}, T2{})
_ = gotots.WriteFromContext(ctx, io.StdOut)
```

Customize prefix of interface name (default is package name):
```go
cfg := &gotots.Config{IndentWithTabs: true}
cfg.FieldPackageNameToPrefix = func(pkg string) string {
	return "Rolancia"
}
cfg.Init()
collect(t, Employee{}, cfg)
```
```typescript
// generated ts
export interface RolanciaParent {
    // ...
}

export interface RolanciaEmployee {
    // ...
}
```