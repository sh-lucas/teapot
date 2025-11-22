# Mug Features

Mug provides several powerful features to streamline backend development in Go.

## ShortBrew

`ShortBrew` is a high-level generic struct designed for handlers that need both Authentication and a JSON Request Body. It automatically handles:
- **Authentication**: Validates the Bearer token and populates the `Auth` field.
- **Body Parsing**: Unmarshals the JSON request body into the `Body` field.

### Usage
```go
type CreateUserRequest struct {
    Username string `json:"username"`
    Email    string `json:"email"`
}

type CreateUserHandler struct {
    mug.ShortBrew[CreateUserRequest, jwt.RegisteredClaims]
}

// The handler function receives the populated struct
func CreateUser(req CreateUserHandler) (int, *UserResponse) {
    // Access Auth info
    fmt.Println("User ID:", req.Auth.Subject)
    
    // Access Body
    fmt.Println("Creating user:", req.Body.Username)
    
    return http.StatusCreated, &UserResponse{Status: "Created"}
}
```

## Espresso

`Espresso` is a lower-level struct that gives you direct access to the `http.ResponseWriter` and `http.Request` while still supporting Mug's mixin system. Use this when you need more control or don't fit the standard `ShortBrew` pattern.

### Usage
```go
type CustomHandler struct {
    mug.Espresso
    // You can still mix in Auth or Body if needed
    mug.JsonBodyT[MyBody]
}

func HandleCustom(req CustomHandler) (int, *Response) {
    // Access raw Writer and Request via .C
    req.C.Writer.Header().Set("X-Custom-Header", "Coffee")
    
    // Access mixed-in body
    fmt.Println(req.Body.SomeField)
    
    return http.StatusOK, &Response{}
}
```

## Struct Inputs

Mug supports strongly-typed handlers. Instead of the traditional `w http.ResponseWriter, r *http.Request` signature, you can define handlers that take a struct as input and return a status code and a response struct.

### Benefits
- **Automatic Validation**: If your input struct uses `validator` tags, Mug will automatically validate the request before calling your handler.
- **Type Safety**: You work with Go structs, not raw JSON.
- **Swagger Generation**: Mug automatically generates OpenAPI/Swagger documentation based on your input and output structs.

### Example
```go
func MyHandler(req MyInputStruct) (int, *MyOutputStruct) {
    // req is already validated and populated
    return 200, &MyOutputStruct{...}
}
```

## Default Routing

Mug uses a code-generation approach for routing. It scans your `handlers` directory for functions annotated with `// mug:handler`.

### How it works
1.  Create a function in the `handlers` directory (or subdirectories).
2.  Add the `// mug:handler <METHOD> <PATH>` comment above the function.
3.  Run `mug gen` (or `mug watch`).
4.  Mug generates the routing code in `./cup/router/router.go`.

### Example
```go
// mug:handler POST /users/create
func CreateUser(req CreateUserRequest) (int, *UserResponse) {
    // ...
}
```

This approach keeps your routing configuration right next to your handler logic, making it easy to see what endpoint triggers which function.

## Swagger / OpenAPI Generation

Mug automatically generates Swagger/OpenAPI documentation for your API. It correctly handles:

- **ShortBrew**: The `Body` field is automatically extracted and documented as the request body.
- **Espresso**: When used with `JsonBodyT`, the body type is correctly identified and documented.
- **Struct Inputs**: The input struct is used to generate the request schema.

You can access the Swagger UI at `/docs` and the raw JSON spec at `/swagger.json`.
