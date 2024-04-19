package azure

import (
	"context"
	"strings"

	"github.com/turbot/steampipe-plugin-sdk/v5/grpc/proto"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin/transform"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/sql/armsql"
)

//// TABLE DEFINITION

func tableAzureSQLServer(_ context.Context) *plugin.Table {
	return &plugin.Table{
		Name:        "azure_sql_server",
		Description: "Azure SQL Server",
		Get: &plugin.GetConfig{
			KeyColumns: plugin.AllColumns([]string{"name", "resource_group"}),
			Hydrate:    getSQLServer,
			IgnoreConfig: &plugin.IgnoreConfig{
				ShouldIgnoreErrorFunc: isNotFoundError([]string{"ResourceNotFound", "ResourceGroupNotFound", "404", "InvalidApiVersionParameter"}),
			},
		},
		List: &plugin.ListConfig{
			Hydrate: listSQLServer,
		},
		Columns: azureColumns([]*plugin.Column{
			{
				Name:        "name",
				Description: "The friendly name that identifies the SQL server.",
				Type:        proto.ColumnType_STRING,
			},
			{
				Name:        "id",
				Description: "Contains ID to identify a SQL server uniquely.",
				Type:        proto.ColumnType_STRING,
				Transform:   transform.FromGo(),
			},
			{
				Name:        "type",
				Description: "The resource type of the SQL server.",
				Type:        proto.ColumnType_STRING,
			},
			{
				Name:        "state",
				Description: "The state of the server.",
				Type:        proto.ColumnType_STRING,
				Transform:   transform.FromField("ServerProperties.State"),
			},
			{
				Name:        "kind",
				Description: "The Kind of sql server.",
				Type:        proto.ColumnType_STRING,
			},
			{
				Name:        "location",
				Description: "The resource location.",
				Type:        proto.ColumnType_STRING,
			},
			{
				Name:        "administrator_login",
				Description: "Specifies the username of the administrator for this server.",
				Type:        proto.ColumnType_STRING,
				Transform:   transform.FromField("ServerProperties.AdministratorLogin"),
			},
			{
				Name:        "administrator_login_password",
				Description: "The administrator login password.",
				Type:        proto.ColumnType_STRING,
				Transform:   transform.FromField("ServerProperties.AdministratorLoginPassword"),
			},
			{
				Name:        "minimal_tls_version",
				Description: "Minimal TLS version. Allowed values: '1.0', '1.1', '1.2'.",
				Type:        proto.ColumnType_STRING,
				Transform:   transform.FromField("ServerProperties.MinimalTLSVersion"),
			},
			{
				Name:        "public_network_access",
				Description: "Whether or not public endpoint access is allowed for this server.",
				Type:        proto.ColumnType_STRING,
				Transform:   transform.FromField("ServerProperties.PublicNetworkAccess"),
			},
			{
				Name:        "version",
				Description: "The version of the server.",
				Type:        proto.ColumnType_STRING,
				Transform:   transform.FromField("ServerProperties.Version"),
			},
			{
				Name:        "fully_qualified_domain_name",
				Description: "The fully qualified domain name of the server.",
				Type:        proto.ColumnType_STRING,
				Transform:   transform.FromField("ServerProperties.FullyQualifiedDomainName"),
			},
			{
				Name:        "server_audit_policy",
				Description: "Specifies the audit policy configuration for server.",
				Type:        proto.ColumnType_JSON,
				Hydrate:     getSQLServerAuditPolicy,
				Transform:   transform.FromValue(),
			},
			{
				Name:        "server_security_alert_policy",
				Description: "Specifies the security alert policy configuration for server.",
				Type:        proto.ColumnType_JSON,
				Hydrate:     getSQLServerSecurityAlertPolicy,
				Transform:   transform.FromValue(),
			},
			{
				Name:        "server_azure_ad_administrator",
				Description: "Specifies the active directory administrator.",
				Type:        proto.ColumnType_JSON,
				Hydrate:     getSQLServerAzureADAdministrator,
				Transform:   transform.FromValue(),
			},
			{
				Name:        "server_vulnerability_assessment",
				Description: "Specifies the server's vulnerability assessment.",
				Type:        proto.ColumnType_JSON,
				Hydrate:     getSQLServerVulnerabilityAssessment,
				Transform:   transform.FromValue(),
			},
			{
				Name:        "firewall_rules",
				Description: "A list of firewall rules fro this server.",
				Type:        proto.ColumnType_JSON,
				Hydrate:     listSQLServerFirewallRules,
				Transform:   transform.FromValue(),
			},
			{
				Name:        "encryption_protector",
				Description: "The server encryption protector.",
				Type:        proto.ColumnType_JSON,
				Hydrate:     getSQLServerEncryptionProtector,
				Transform:   transform.FromValue(),
			},
			{
				Name:        "private_endpoint_connections",
				Description: "The private endpoint connections of the sql server.",
				Type:        proto.ColumnType_JSON,
				Hydrate:     listSQLServerPrivateEndpointConnections,
				Transform:   transform.FromValue(),
			},
			{
				Name:        "tags_src",
				Description: "Specifies the set of tags attached to the server.",
				Type:        proto.ColumnType_JSON,
				Transform:   transform.FromField("Tags"),
			},
			{
				Name:        "virtual_network_rules",
				Description: "A list of virtual network rules for this server.",
				Type:        proto.ColumnType_JSON,
				Hydrate:     listSQLServerVirtualNetworkRules,
				Transform:   transform.FromValue(),
			},

			// Steampipe standard columns
			{
				Name:        "title",
				Description: ColumnDescriptionTitle,
				Type:        proto.ColumnType_STRING,
				Transform:   transform.FromField("Name"),
			},
			{
				Name:        "tags",
				Description: ColumnDescriptionTags,
				Type:        proto.ColumnType_JSON,
			},
			{
				Name:        "akas",
				Description: ColumnDescriptionAkas,
				Type:        proto.ColumnType_JSON,
				Transform:   transform.FromField("ID").Transform(idToAkas),
			},

			// Azure standard columns
			{
				Name:        "region",
				Description: ColumnDescriptionRegion,
				Type:        proto.ColumnType_STRING,
				Transform:   transform.FromField("Location").Transform(toLower),
			},
			{
				Name:        "resource_group",
				Description: ColumnDescriptionResourceGroup,
				Type:        proto.ColumnType_STRING,
				Transform:   transform.FromField("ID").Transform(extractResourceGroupFromID),
			},
		}),
	}
}

type PrivateConnectionInfo struct {
	PrivateEndpointConnectionId                      string
	PrivateEndpointId                                string
	PrivateEndpointConnectionName                    string
	PrivateEndpointConnectionType                    string
	PrivateLinkServiceConnectionStateStatus          string
	PrivateLinkServiceConnectionStateDescription     string
	PrivateLinkServiceConnectionStateActionsRequired string
	ProvisioningState                                string
}

//// LIST FUNCTION

func listSQLServer(ctx context.Context, d *plugin.QueryData, _ *plugin.HydrateData) (interface{}, error) {
	session, err := GetNewSessionUpdated(ctx, d)
	if err != nil {
		return nil, err
	}
	client, err := armsql.NewServersClient(session.SubscriptionID, session.Cred, nil)
	if err != nil {
		return nil, err
	}

	pager := client.NewListPager(nil)
	for pager.More() {
		result, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, server := range result.Value {
			d.StreamListItem(ctx, *server)
			// Check if context has been cancelled or if the limit has been hit (if specified)
			// if there is a limit, it will return the number of rows required to reach this limit
			if d.RowsRemaining(ctx) == 0 {
				return nil, nil
			}
		}
	}

	return nil, err
}

//// HYDRATE FUNCTIONS

func getSQLServer(ctx context.Context, d *plugin.QueryData, h *plugin.HydrateData) (interface{}, error) {
	plugin.Logger(ctx).Trace("getSQLServer")

	name := d.EqualsQuals["name"].GetStringValue()
	resourceGroup := d.EqualsQuals["resource_group"].GetStringValue()

	session, err := GetNewSessionUpdated(ctx, d)
	if err != nil {
		return nil, err
	}
	client, err := armsql.NewServersClient(session.SubscriptionID, session.Cred, nil)
	if err != nil {
		return nil, err
	}

	op, err := client.Get(ctx, resourceGroup, name, nil)
	if err != nil {
		return nil, err
	}

	// In some cases resource does not give any notFound error
	// instead of notFound error, it returns empty data
	if op.ID != nil {
		return op.Server, nil
	}

	return nil, nil
}

func getSQLServerAuditPolicy(ctx context.Context, d *plugin.QueryData, h *plugin.HydrateData) (interface{}, error) {
	plugin.Logger(ctx).Trace("getSQLServerAuditPolicy")

	server := h.Item.(armsql.Server)
	serverName := *server.Name
	resourceGroupName := strings.Split(string(*server.ID), "/")[4]

	session, err := GetNewSessionUpdated(ctx, d)
	if err != nil {
		return nil, err
	}
	client, err := armsql.NewServerBlobAuditingPoliciesClient(session.SubscriptionID, session.Cred, nil)
	if err != nil {
		return nil, err
	}

	var auditPolicies []*armsql.ServerBlobAuditingPolicy
	pager := client.NewListByServerPager(resourceGroupName, serverName, nil)
	for pager.More() {
		result, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		auditPolicies = append(auditPolicies, result.Value...)
	}

	// // If we return the API response directly, the output only gives
	// // the contents of ServerBlobAuditingPolicyProperties
	// var auditPolicies []map[string]interface{}
	// for _, i := range op.Values() {
	// 	objectMap := make(map[string]interface{})
	// 	if i.ID != nil {
	// 		objectMap["id"] = i.ID
	// 	}
	// 	if i.Name != nil {
	// 		objectMap["name"] = i.Name
	// 	}
	// 	if i.Type != nil {
	// 		objectMap["type"] = i.Type
	// 	}
	// 	if i.ServerBlobAuditingPolicyProperties != nil {
	// 		objectMap["properties"] = i.ServerBlobAuditingPolicyProperties
	// 	}
	// 	auditPolicies = append(auditPolicies, objectMap)
	// }
	return auditPolicies, nil
}

func listSQLServerPrivateEndpointConnections(ctx context.Context, d *plugin.QueryData, h *plugin.HydrateData) (interface{}, error) {
	plugin.Logger(ctx).Trace("listSQLServerPrivateEndpointConnections")
	server := h.Item.(armsql.Server)
	serverName := *server.Name
	resourceGroupName := strings.Split(string(*server.ID), "/")[4]

	session, err := GetNewSessionUpdated(ctx, d)
	if err != nil {
		return nil, err
	}
	client, err := armsql.NewPrivateEndpointConnectionsClient(session.SubscriptionID, session.Cred, nil)
	if err != nil {
		return nil, err
	}

	var privateEndpointConnections []*armsql.PrivateEndpointConnection
	pager := client.NewListByServerPager(resourceGroupName, serverName, nil)
	for pager.More() {
		result, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		privateEndpointConnections = append(privateEndpointConnections, result.Value...)
	}

	return privateEndpointConnections, nil
}

func getSQLServerSecurityAlertPolicy(ctx context.Context, d *plugin.QueryData, h *plugin.HydrateData) (interface{}, error) {
	plugin.Logger(ctx).Trace("getSQLServerSecurityAlertPolicy")
	server := h.Item.(armsql.Server)
	serverName := *server.Name
	resourceGroupName := strings.Split(string(*server.ID), "/")[4]

	session, err := GetNewSessionUpdated(ctx, d)
	if err != nil {
		return nil, err
	}
	client, err := armsql.NewServerSecurityAlertPoliciesClient(session.SubscriptionID, session.Cred, nil)
	if err != nil {
		return nil, err
	}

	var securityAlertPolicies []*armsql.ServerSecurityAlertPolicy
	pager := client.NewListByServerPager(resourceGroupName, serverName, nil)
	for pager.More() {
		result, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		securityAlertPolicies = append(securityAlertPolicies, result.Value...)
	}

	return securityAlertPolicies, nil
}

func getSQLServerAzureADAdministrator(ctx context.Context, d *plugin.QueryData, h *plugin.HydrateData) (interface{}, error) {
	plugin.Logger(ctx).Trace("getSQLServerAzureADAdministrator")
	server := h.Item.(armsql.Server)
	serverName := *server.Name
	resourceGroupName := strings.Split(string(*server.ID), "/")[4]

	session, err := GetNewSessionUpdated(ctx, d)
	if err != nil {
		return nil, err
	}
	client, err := armsql.NewServerAzureADAdministratorsClient(session.SubscriptionID, session.Cred, nil)
	if err != nil {
		return nil, err
	}

	var serverAdministrators []*armsql.ServerAzureADAdministrator
	pager := client.NewListByServerPager(resourceGroupName, serverName, nil)
	for pager.More() {
		result, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		serverAdministrators = append(serverAdministrators, result.Value...)
	}

	return serverAdministrators, nil
}

func getSQLServerEncryptionProtector(ctx context.Context, d *plugin.QueryData, h *plugin.HydrateData) (interface{}, error) {
	plugin.Logger(ctx).Trace("getSQLServerEncryptionProtector")
	server := h.Item.(armsql.Server)
	serverName := *server.Name
	resourceGroupName := strings.Split(string(*server.ID), "/")[4]

	session, err := GetNewSessionUpdated(ctx, d)
	if err != nil {
		return nil, err
	}
	client, err := armsql.NewEncryptionProtectorsClient(session.SubscriptionID, session.Cred, nil)
	if err != nil {
		return nil, err
	}

	var encryptionProtectors []*armsql.EncryptionProtector
	pager := client.NewListByServerPager(resourceGroupName, serverName, nil)
	for pager.More() {
		result, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		encryptionProtectors = append(encryptionProtectors, result.Value...)
	}

	return encryptionProtectors, nil
}

func getSQLServerVulnerabilityAssessment(ctx context.Context, d *plugin.QueryData, h *plugin.HydrateData) (interface{}, error) {
	plugin.Logger(ctx).Trace("getSQLServerVulnerabilityAssessment")
	server := h.Item.(armsql.Server)
	serverName := *server.Name
	resourceGroupName := strings.Split(string(*server.ID), "/")[4]

	session, err := GetNewSessionUpdated(ctx, d)
	if err != nil {
		return nil, err
	}
	client, err := armsql.NewServerVulnerabilityAssessmentsClient(session.SubscriptionID, session.Cred, nil)
	if err != nil {
		return nil, err
	}

	var vulnerabilityAssessments []*armsql.ServerVulnerabilityAssessment
	pager := client.NewListByServerPager(resourceGroupName, serverName, nil)
	for pager.More() {
		result, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		vulnerabilityAssessments = append(vulnerabilityAssessments, result.Value...)
	}

	return vulnerabilityAssessments, nil
}

func listSQLServerFirewallRules(ctx context.Context, d *plugin.QueryData, h *plugin.HydrateData) (interface{}, error) {
	plugin.Logger(ctx).Trace("listSQLServerFirewallRules")
	server := h.Item.(armsql.Server)
	serverName := *server.Name
	resourceGroupName := strings.Split(string(*server.ID), "/")[4]

	session, err := GetNewSessionUpdated(ctx, d)
	if err != nil {
		return nil, err
	}
	client, err := armsql.NewFirewallRulesClient(session.SubscriptionID, session.Cred, nil)
	if err != nil {
		return nil, err
	}

	var firewallRules []*armsql.FirewallRule
	pager := client.NewListByServerPager(resourceGroupName, serverName, nil)
	for pager.More() {
		result, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		firewallRules = append(firewallRules, result.Value...)
	}

	return firewallRules, nil
}

func listSQLServerVirtualNetworkRules(ctx context.Context, d *plugin.QueryData, h *plugin.HydrateData) (interface{}, error) {
	plugin.Logger(ctx).Trace("listSQLServerVirtualNetworkRules")
	var resourceGroupName, serverName string
	if h.Item != nil {
		switch item := h.Item.(type) {
		case *armsql.Server:
			serverName = *item.Name
			resourceGroupName = strings.Split(string(*item.ID), "/")[4]
		case armsql.ServersClientGetResponse:
			serverName = *item.Name
			resourceGroupName = strings.Split(string(*item.ID), "/")[4]
		}
	}

	session, err := GetNewSessionUpdated(ctx, d)
	if err != nil {
		return nil, err
	}
	client, err := armsql.NewVirtualNetworkRulesClient(session.SubscriptionID, session.Cred, nil)
	if err != nil {
		return nil, err
	}

	var networkRules []*armsql.VirtualNetworkRule
	pager := client.NewListByServerPager(resourceGroupName, serverName, nil)
	for pager.More() {
		result, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		networkRules = append(networkRules, result.Value...)
	}

	return networkRules, nil
}

// func networkRuleMap(rule sql.VirtualNetworkRule) map[string]interface{} {
// 	objectMap := make(map[string]interface{})
// 	if rule.ID != nil {
// 		objectMap["id"] = rule.ID
// 	}
// 	if rule.Name != nil {
// 		objectMap["name"] = rule.Name
// 	}
// 	if rule.Type != nil {
// 		objectMap["type"] = rule.Type
// 	}
// 	if rule.VirtualNetworkRuleProperties != nil {
// 		objectMap["properties"] = rule.VirtualNetworkRuleProperties
// 	}
// 	return objectMap
// }

// If we return the API response directly, the output will not give
// all the contents of PrivateEndpointConnection
// func privateEndpointConnectionMap(conn sql.PrivateEndpointConnection) PrivateConnectionInfo {
// 	var connection PrivateConnectionInfo
// 	if conn.ID != nil {
// 		connection.PrivateEndpointConnectionId = *conn.ID
// 	}
// 	if conn.Name != nil {
// 		connection.PrivateEndpointConnectionName = *conn.Name
// 	}
// 	if conn.Type != nil {
// 		connection.PrivateEndpointConnectionType = *conn.Type
// 	}
// 	if conn.PrivateEndpointConnectionProperties != nil {
// 		if conn.PrivateEndpoint != nil {
// 			if conn.PrivateEndpoint.ID != nil {
// 				connection.PrivateEndpointId = *conn.PrivateEndpoint.ID
// 			}
// 		}
// 		if conn.PrivateLinkServiceConnectionState != nil {
// 			if conn.PrivateLinkServiceConnectionState.ActionsRequired != "" {
// 				connection.PrivateLinkServiceConnectionStateActionsRequired = string(conn.PrivateLinkServiceConnectionState.ActionsRequired)
// 			}
// 			if conn.PrivateLinkServiceConnectionState.Status != "" {
// 				connection.PrivateLinkServiceConnectionStateStatus = string(conn.PrivateLinkServiceConnectionState.Status)
// 			}
// 			if conn.PrivateLinkServiceConnectionState.Description != nil {
// 				connection.PrivateLinkServiceConnectionStateDescription = *conn.PrivateLinkServiceConnectionState.Description
// 			}
// 		}
// 		if conn.ProvisioningState != "" {
// 			connection.ProvisioningState = string(conn.ProvisioningState)
// 		}
// 	}

// 	return connection
// }
