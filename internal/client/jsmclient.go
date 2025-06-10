package client

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"

	jira "github.com/andygrunwald/go-jira"
	"github.com/hasura/go-graphql-client"
)

// error message conflict
const ErrRevisionConflict = "Specified revision was incorrect"

type Service struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	Description     *string `json:"description,omitempty"`
	Revision        string  `json:"revision"`
	TierLevel       int     `json:"tierLevel"`
	ApplicationType string  `json:"applicationType,omitempty"` // Optional, e.g., "APPLICATIONS", "BUSINESS_SERVICES"
	TierID          string  `json:"tierId,omitempty"`
}

type CreateServiceRequest struct {
	Name        string
	Description string
	TierLevel   int
	ServiceType string
	TeamARNs    []string
	CloudID     string
}

type JSMClient struct {
	GraphQLClient *graphql.Client
	JiraClient    *jira.Client
	CloudID       string
}

type JSMConfig struct {
	GraphQLURL string
	RestURL    string
	Token      string
	Username   string
	CloudID    string
}

type CreateDevOpsServiceInput struct {
	Name        string `json:"name"`
	CloudID     string `json:"cloudId"`
	Description string `json:"description"`
	ServiceTier struct {
		Level int `json:"level"`
	} `json:"serviceTier"`
	ServiceType struct {
		Key string `json:"key"`
	} `json:"serviceType"`
	Properties []struct {
		Key   string `json:"key"`
		Value struct {
			Teams []string `json:"teams"`
		} `json:"value"`
	} `json:"properties"`
}

type UpdateServiceRequest struct {
	ID          string // Full ARI
	Name        string
	Revision    string
	ServiceType string // e.g., "APPLICATIONS", "BUSINESS_SERVICES"
	TierID      string
	TeamARNs    []string
	Description string
}

type UpdateDevOpsServiceInput struct {
	ID          string                  `json:"id"`
	Description string                  `json:"description"`
	Name        string                  `json:"name"`
	Revision    string                  `json:"revision"`
	ServiceTier string                  `json:"serviceTier"`
	Properties  []updateServiceProperty `json:"properties"`
}

type updateServiceProperty struct {
	Key   string               `json:"key"`
	Value updateResponderValue `json:"value"`
}

type updateResponderValue struct {
	Teams []string `json:"teams"`
}

func NewJSMClient(config JSMConfig) (*JSMClient, error) {
	if config.GraphQLURL == "" || config.RestURL == "" || config.Token == "" || config.Username == "" || config.CloudID == "" {
		return nil, errors.New("invalid JSM configuration: all fields must be provided")
	}

	tp := jira.BasicAuthTransport{
		Username: config.Username,
		Password: config.Token,
	}

	if !strings.HasSuffix(config.RestURL, "/") {
		config.RestURL += "/"
	}

	jiraClient, err := jira.NewClient(tp.Client(), fmt.Sprintf("%s/v1/", config.RestURL))
	if err != nil {
		return nil, err
	}

	graphqlClient := graphql.NewClient(config.GraphQLURL, http.DefaultClient).WithRequestModifier(func(req *http.Request) {
		req.Header.Set("Authorization", "Basic "+basicAuth(config.Username, config.Token))
	})

	return &JSMClient{
		GraphQLClient: graphqlClient,
		JiraClient:    jiraClient,
		CloudID:       config.CloudID,
	}, nil
}

func basicAuth(username, password string) string {
	return base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
}

// GetServiceByName retrieves a JSM service by its name.
// It's a graphql api, since JSM services are not available via the REST API.
func (c *JSMClient) GetServiceByName(ctx context.Context, name string) (*Service, error) {
	var query struct {
		DevOpsServices struct {
			Edges []struct {
				Node struct {
					ID       string
					Name     string
					Revision string
				}
			}
		} `graphql:"devOpsServices(cloudId: $cloudId, filter: { nameContains: $name })"`
	}

	variables := map[string]any{
		"cloudId": graphql.String(c.CloudID),
		"name":    graphql.String(name),
	}
	err := c.GraphQLClient.Query(ctx, &query, variables, graphql.OperationName("GetServiceByName"))

	if err != nil {
		return nil, err
	}

	if len(query.DevOpsServices.Edges) == 0 {
		return nil, nil
	}

	// we expect only one service with the given name, so we can safely return the first one
	serviceNode := query.DevOpsServices.Edges[0].Node
	return &Service{
		ID:       serviceNode.ID,
		Name:     serviceNode.Name,
		Revision: serviceNode.Revision,
	}, nil
}

// CreateService creates a new JSM service with the given specifications.
func (c *JSMClient) CreateService(ctx context.Context, req *CreateServiceRequest) (*Service, error) {
	var mutation struct {
		CreateDevOpsService struct {
			Success bool `json:"success"`
			Errors  []struct {
				Message string `json:"message"`
			} `json:"errors"`
			Service struct {
				ID          string `json:"id"`
				Name        string `json:"name"`
				Revision    string `json:"revision"`
				ServiceTier struct {
					ID    string `json:"id"`
					Level int    `json:"level"`
				} `json:"serviceTier"`
			} `json:"service"`
		} `graphql:"createDevOpsService(input: $input)"`
	}
	input := CreateDevOpsServiceInput{
		Name:    req.Name,
		CloudID: c.CloudID,
	}
	input.ServiceTier.Level = req.TierLevel
	input.ServiceType.Key = req.ServiceType
	input.Description = req.Description

	// as for now we support only the reponders property with teams
	input.Properties = []struct {
		Key   string `json:"key"`
		Value struct {
			Teams []string `json:"teams"`
		} `json:"value"`
	}{
		{
			Key: "responders",
			Value: struct {
				Teams []string `json:"teams"`
			}{
				Teams: req.TeamARNs,
			},
		},
	}

	// pass the input to the mutation
	variables := map[string]any{
		"input": input,
	}

	err := c.GraphQLClient.Mutate(ctx, &mutation, variables, graphql.OperationName("CreateDevOpsService"))
	if err != nil {
		return nil, fmt.Errorf("mutation failed: %w", err)
	}

	if !mutation.CreateDevOpsService.Success {
		var messages []string
		for _, e := range mutation.CreateDevOpsService.Errors {
			messages = append(messages, e.Message)
		}
		return nil, fmt.Errorf("jsm service creation failed: %s", strings.Join(messages, "; "))
	}

	svc := mutation.CreateDevOpsService.Service
	return &Service{
		ID:        svc.ID,
		Name:      svc.Name,
		Revision:  svc.Revision,
		TierLevel: svc.ServiceTier.Level,
		TierID:    svc.ServiceTier.ID,
	}, nil
}

func (c *JSMClient) GetTierIDByLevel(ctx context.Context, level int) (string, error) {
	var query struct {
		DevOpsServiceTiers []struct {
			ID    string `json:"id"`
			Level int    `json:"level"`
		} `graphql:"devOpsServiceTiers(cloudId: $cloudId)"`
	}

	variables := map[string]any{
		"cloudId": graphql.String(c.CloudID),
	}

	err := c.GraphQLClient.Query(ctx, &query, variables, graphql.OperationName("GetTierIDByLevel"))
	if err != nil {
		return "", fmt.Errorf("failed to query service tiers: %w", err)
	}

	for _, tier := range query.DevOpsServiceTiers {
		if tier.Level == level {
			return tier.ID, nil
		}
	}

	return "", fmt.Errorf("no service tier found for level %d", level)
}

// UpdateService updates an existing JSM service with the given specifications.
func (c *JSMClient) UpdateService(ctx context.Context, req *UpdateServiceRequest) (*Service, error) {
	var mutation struct {
		UpdateDevOpsService struct {
			Success bool `json:"success"`
			Errors  []struct {
				Message string `json:"message"`
			} `json:"errors"`
			Service struct {
				ID          string `json:"id"`
				Name        string `json:"name"`
				Revision    string `json:"revision"`
				ServiceTier struct {
					ID    string `json:"id"`
					Level int    `json:"level"`
				} `json:"serviceTier"`
				ServiceType struct {
					Key string `json:"key"`
				} `json:"serviceType"`
			} `json:"service"`
		} `graphql:"updateDevOpsService(input: $input)"`
	}

	input := UpdateDevOpsServiceInput{
		ID:          req.ID,
		Name:        req.Name,
		Revision:    req.Revision,
		ServiceTier: req.TierID,
		Description: req.Description,
		Properties: []updateServiceProperty{
			{
				Key: "responders",
				Value: updateResponderValue{
					Teams: req.TeamARNs,
				},
			},
		},
	}

	variables := map[string]any{
		"input": input,
	}

	err := c.GraphQLClient.Mutate(ctx, &mutation, variables, graphql.OperationName("UpdateDevOpsService"))
	if err != nil {
		return nil, fmt.Errorf("mutation failed: %w", err)
	}

	if !mutation.UpdateDevOpsService.Success {
		var messages []string
		for _, e := range mutation.UpdateDevOpsService.Errors {
			messages = append(messages, e.Message)
		}
		return nil, fmt.Errorf("jsm service update failed: %s", strings.Join(messages, "; "))
	}

	svc := mutation.UpdateDevOpsService.Service
	return &Service{
		ID:              svc.ID,
		Name:            svc.Name,
		Revision:        svc.Revision,
		TierLevel:       svc.ServiceTier.Level,
		TierID:          svc.ServiceTier.ID,
		ApplicationType: svc.ServiceType.Key,
	}, nil
}

func (c *JSMClient) IsRevisionConflict(err error) bool {
	return strings.Contains(err.Error(), ErrRevisionConflict)
}

func (c *JSMClient) CreateOpsgenieTeamRelationship(ctx context.Context, serviceID, teamID string) (string, error) {
	var mutation struct {
		CreateDevOpsServiceAndOpsgenieTeamRelationship struct {
			Success bool `json:"success"`
			Errors  []struct {
				Message string `json:"message"`
			} `json:"errors"`
			ServiceAndOpsgenieTeamRelationship struct {
				ID string `json:"id"`
			} `json:"serviceAndOpsgenieTeamRelationship"`
		} `graphql:"createDevOpsServiceAndOpsgenieTeamRelationship(input: {cloudId: $cloudId, serviceId: $serviceId, opsgenieTeamId: $teamId})"`
	}

	variables := map[string]any{
		"cloudId":   graphql.ID(c.CloudID),
		"serviceId": graphql.ID(serviceID),
		"teamId":    graphql.ID(teamID),
	}

	err := c.GraphQLClient.Mutate(ctx, &mutation, variables, graphql.OperationName("CreateDevOpsServiceAndOpsgenieTeamRelationship"))
	if err != nil {
		return "", fmt.Errorf("relationship mutation failed: %w", err)
	}

	if !mutation.CreateDevOpsServiceAndOpsgenieTeamRelationship.Success {
		var messages []string
		for _, e := range mutation.CreateDevOpsServiceAndOpsgenieTeamRelationship.Errors {
			messages = append(messages, e.Message)
		}
		return "", fmt.Errorf("failed to create relationship: %s", strings.Join(messages, "; "))
	}

	return mutation.CreateDevOpsServiceAndOpsgenieTeamRelationship.ServiceAndOpsgenieTeamRelationship.ID, nil
}

func (c *JSMClient) GetOpsgenieTeamIDByName(ctx context.Context, name string) (string, error) {
	var query struct {
		Opsgenie struct {
			AllOpsgenieTeams struct {
				Edges []struct {
					Node struct {
						ID   string `json:"id"`
						Name string `json:"name"`
					} `json:"node"`
				} `json:"edges"`
			} `graphql:"allOpsgenieTeams(cloudId: $cloudId)"`
		} `graphql:"opsgenie"`
	}

	variables := map[string]any{
		"cloudId": graphql.ID(c.CloudID),
	}

	err := c.GraphQLClient.Query(ctx, &query, variables, graphql.OperationName("ResolveOpsgenieTeamIDByName"))
	if err != nil {
		return "", fmt.Errorf("failed to query opsgenie teams: %w", err)
	}

	for _, edge := range query.Opsgenie.AllOpsgenieTeams.Edges {
		if edge.Node.Name == name {
			return edge.Node.ID, nil
		}
	}

	return "", fmt.Errorf("opsgenie team with name %q not found", name)
}
