from pony.orm import db_session, select

from cloudfirewall.server.plugins.common.db import BaseDBService
from cloudfirewall.server.plugins.nftables.dto import CreateFirewallRequest, FirewallRuleRequest
from cloudfirewall.server.plugins.nftables.entities import SecurityGroup, SecurityGroupRule


class DatabaseService(BaseDBService):

    def __init__(self):
        super(DatabaseService, self).__init__()

    @db_session
    def create_firewall_group(self, create_request: CreateFirewallRequest):
        SecurityGroup.create(create_request)

    @db_session
    def list_firewall_groups(self):
        groups = select(sg for sg in SecurityGroup)[:]
        return groups

    @db_session
    def get_firewall_group(self, group_id):
        return SecurityGroup[group_id]

    def update_firewall_group(self):
        pass

    def delete_firewall_group(self):
        pass

    @db_session
    def get_firewall_rule(self, rule_id):
        return SecurityGroup[rule_id]

    def add_rule_in_group(self, group, rule):
        try:
            SecurityGroupRule.create(group, rule)
        except Exception as ex:
            self.logger.exception(ex)

    @db_session
    def update_rule_in_group(self, rule_ro: SecurityGroupRule, rule_request: FirewallRuleRequest):
        try:
            rule_rw = SecurityGroupRule[rule_ro.id]
            rule_rw.update_rule(rule_request)
        except Exception as ex:
            self.logger.exception(ex)

    @db_session
    def delete_rule_in_group(self, rule_ro: SecurityGroupRule):
        try:
            rule_rw = SecurityGroupRule[rule_ro.id]
            rule_rw.delete()
        except Exception as ex:
            self.logger.exception(ex)