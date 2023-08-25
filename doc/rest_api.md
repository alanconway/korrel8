


# REST API for korrel8r
  

## Informations

### Version

v1alpha1

### Contact

  https://github.com/korrel8r/korrel8r

## Content negotiation

### URI Schemes
  * http

### Consumes
  * application/json

### Produces
  * application/json

## All endpoints

###  configuration

| Method  | URI     | Name   | Summary |
|---------|---------|--------|---------|
| GET | /api/v1alpha1/domains | [get domains](#get-domains) | List all korrel8r domain names. |
| GET | /api/v1alpha1/stores | [get stores](#get-stores) | List of all store configurations objects. |
| GET | /api/v1alpha1/stores/{domain} | [get stores domain](#get-stores-domain) | List of all store configurations objects for a specific domain. |
  


###  search

| Method  | URI     | Name   | Summary |
|---------|---------|--------|---------|
| POST | /api/v1alpha1/graph/goals | [post graph goals](#post-graph-goals) | Create a correlation graph from start objects to goal queries. |
| POST | /api/v1alpha1/graph/neighbours | [post graph neighbours](#post-graph-neighbours) | Create a correlation graph of neighbours of a start object to a given depth. |
| POST | /api/v1alpha1/list/goals | [post list goals](#post-list-goals) | Generate a list of goal nodes related to a starting point. |
  


## Paths

### <span id="get-domains"></span> List all korrel8r domain names. (*GetDomains*)

```
GET /api/v1alpha1/domains
```

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-domains-200) | OK | OK |  | [schema](#get-domains-200-schema) |

#### Responses


##### <span id="get-domains-200"></span> 200 - OK
Status: OK

###### <span id="get-domains-200-schema"></span> Schema
   
  

[]string

### <span id="get-stores"></span> List of all store configurations objects. (*GetStores*)

```
GET /api/v1alpha1/stores
```

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-stores-200) | OK | OK |  | [schema](#get-stores-200-schema) |

#### Responses


##### <span id="get-stores-200"></span> 200 - OK
Status: OK

###### <span id="get-stores-200-schema"></span> Schema
   
  

[][APIStoreConfig](#api-store-config)

### <span id="get-stores-domain"></span> List of all store configurations objects for a specific domain. (*GetStoresDomain*)

```
GET /api/v1alpha1/stores/{domain}
```

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| domain | `path` | string | `string` |  | ✓ |  | domain	name |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-stores-domain-200) | OK | OK |  | [schema](#get-stores-domain-200-schema) |

#### Responses


##### <span id="get-stores-domain-200"></span> 200 - OK
Status: OK

###### <span id="get-stores-domain-200-schema"></span> Schema
   
  

[][APIStoreConfig](#api-store-config)

### <span id="post-graph-goals"></span> Create a correlation graph from start objects to goal queries. (*PostGraphGoals*)

```
POST /api/v1alpha1/graph/goals
```

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| withRules | `query` | boolean | `bool` |  |  |  | include rules in graph edges |
| start | `body` | [APIGoalsRequest](#api-goals-request) | `models.APIGoalsRequest` | | ✓ | | search from start to goal classes |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#post-graph-goals-200) | OK | OK |  | [schema](#post-graph-goals-200-schema) |

#### Responses


##### <span id="post-graph-goals-200"></span> 200 - OK
Status: OK

###### <span id="post-graph-goals-200-schema"></span> Schema
   
  

[APIGraph](#api-graph)

### <span id="post-graph-neighbours"></span> Create a correlation graph of neighbours of a start object to a given depth. (*PostGraphNeighbours*)

```
POST /api/v1alpha1/graph/neighbours
```

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| withRules | `query` | boolean | `bool` |  |  |  | include rules in graph edges |
| start | `body` | [APINeighboursRequest](#api-neighbours-request) | `models.APINeighboursRequest` | | ✓ | | search from neighbours |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#post-graph-neighbours-200) | OK | OK |  | [schema](#post-graph-neighbours-200-schema) |

#### Responses


##### <span id="post-graph-neighbours-200"></span> 200 - OK
Status: OK

###### <span id="post-graph-neighbours-200-schema"></span> Schema
   
  

[APIGraph](#api-graph)

### <span id="post-list-goals"></span> Generate a list of goal nodes related to a starting point. (*PostListGoals*)

```
POST /api/v1alpha1/list/goals
```

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| start | `body` | [APIGoalsRequest](#api-goals-request) | `models.APIGoalsRequest` | | ✓ | | search from start to goal classes |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#post-list-goals-200) | OK | OK |  | [schema](#post-list-goals-200-schema) |

#### Responses


##### <span id="post-list-goals-200"></span> 200 - OK
Status: OK

###### <span id="post-list-goals-200-schema"></span> Schema
   
  

[][APINode](#api-node)

## Models

### <span id="api-edge"></span> api.Edge


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| goal | string| `string` |  | | Goal is the class name of the goal node. | `class.domain` |
| rules | [][APIRule](#api-rule)| `[]*APIRule` |  | | Rules is the set of rules followed along this edge (optional). |  |
| start | string| `string` |  | | Start is the class name of the start node. |  |



### <span id="api-goals-request"></span> api.GoalsRequest


> Starting point for a goals search.
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| goals | []string| `[]string` |  | | Goal classes for correlation. | `["class.domain"]` |
| start | [APIGoalsRequest](#api-goals-request)| `APIGoalsRequest` |  | | Start of correlation search. |  |



### <span id="api-graph"></span> api.Graph


> Graph resulting from a correlation search.
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| edges | [][APIEdge](#api-edge)| `[]*APIEdge` |  | |  |  |
| nodes | [][APINode](#api-node)| `[]*APINode` |  | |  |  |



### <span id="api-neighbours-request"></span> api.NeighboursRequest


> Starting point for a neighbours search.
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| depth | integer| `int64` |  | | Max depth of neighbours graph. |  |
| start | [APINeighboursRequest](#api-neighbours-request)| `APINeighboursRequest` |  | | Start of correlation search. |  |



### <span id="api-node"></span> api.Node


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| class | string| `string` |  | | Class is the full name of the class in "CLASS.DOMAIN" form. | `class.domain` |
| count | integer| `int64` |  | | Count of results found for this class, after de-duplication. |  |
| queries | [APINode](#api-node)| `APINode` |  | | Queries yielding results for this class. | `{"querystring":10}` |



### <span id="api-queries"></span> api.Queries


> A set of query strings with counts of results found by the query. Value of -1 means the query was not run so result count is unknown.
  



[APIQueries](#api-queries)

### <span id="api-rule"></span> api.Rule


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| name | string| `string` |  | | Name is an optional descriptive name. |  |
| queries | [APIRule](#api-rule)| `APIRule` |  | | Queries generated while following this rule. | `{"querystring":10}` |



### <span id="api-start"></span> api.Start


> Starting point for correlation.
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| class | string| `string` |  | | Class of starting objects | `class.domain` |
| objects | [interface{}](#interface)| `interface{}` |  | | Objects in JSON form |  |
| query | []string| `[]string` |  | | Queries for starting objects |  |



### <span id="api-store-config"></span> api.StoreConfig


  

[APIStoreConfig](#api-store-config)
