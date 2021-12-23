# Packetframe API

### Routes:
| Route | Method | Description |
| :---- | :----- | :---------- |
| /meta | GET | Get API metadata |
| /user/login | POST | Log a user in |
| /user/signup | POST | Create a new user account |
| /user/logout | POST | Log a user out |
| /user/delete | DELETE | Delete a user account |
| /user/password | POST | Change a user's password |
| /user/info | GET | Get user info |
| /dns/zones | GET | List all DNS zones authorized for a user |
| /dns/zones | POST | Add a new DNS zone |
| /dns/zones | DELETE | Delete a DNS zone |
| /dns/zones/user | PUT | Add a user to a DNS zone |
| /dns/zones/user | DELETE | Remove a user from a DNS zone |
| /dns/records/:id | GET | List DNS records for a zone |
| /dns/records | POST | Add a DNS record to a zone |
| /dns/records | DELETE | Delete a DNS record from a zone |
| /dns/records | PUT | Update a DNS record |
| /admin/user/list | GET | Get a list of all users |
| /admin/user/groups | PUT | Add a group to a user |
| /admin/user/groups | DELETE | Remove a group from a user |
| /admin/user/impersonate | POST | Log in as another user |

