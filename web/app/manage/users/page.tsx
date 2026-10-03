"use client";

import { ApiError } from "@/api/base";
import * as usersApi from "@/api/users";
import { PanelTable } from "@/assets/components/PanelTable";
import {
  ChangePasswordModal,
  CreateUserModal,
} from "@/assets/modals/UserFormModals";
import { useApiErrorMessage, useTranslation } from "@/contexts/LocaleContext";
import type { User } from "@/types/user";
import { App, Button, Input, Space, Table, Tag } from "antd";
import type { ColumnsType, TablePaginationConfig } from "antd/es/table";
import { useCallback, useEffect, useState } from "react";

const PAGE_SIZE = 20;

export default function ManageUsersPage() {
  const { message, modal } = App.useApp();
  const { t } = useTranslation();
  const formatApiError = useApiErrorMessage();
  const [data, setData] = useState<User[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState("");
  const [searchInput, setSearchInput] = useState("");
  const [loading, setLoading] = useState(false);

  const [createOpen, setCreateOpen] = useState(false);
  const [createLoading, setCreateLoading] = useState(false);

  const [passwordUser, setPasswordUser] = useState<User | null>(null);
  const [passwordLoading, setPasswordLoading] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await usersApi.listUsers({
        page,
        page_size: PAGE_SIZE,
        search: search || undefined,
      });
      setData(res.results);
      setTotal(res.count);
    } catch (e) {
      message.error(formatApiError(e, t("users.loadFailed")));
    } finally {
      setLoading(false);
    }
  }, [formatApiError, message, page, search, t]);

  useEffect(() => {
    void load();
  }, [load]);

  const handleCreate = async (values: { username: string; password: string }) => {
    setCreateLoading(true);
    try {
      await usersApi.createUser(values);
      message.success(t("users.created"));
      setCreateOpen(false);
      setPage(1);
      const res = await usersApi.listUsers({
        page: 1,
        page_size: PAGE_SIZE,
        search: search || undefined,
      });
      setData(res.results);
      setTotal(res.count);
    } catch (e) {
      message.error(formatApiError(e, t("users.createFailed")));
    } finally {
      setCreateLoading(false);
    }
  };

  const handleChangePassword = async (values: { password: string }) => {
    if (!passwordUser) {
      return;
    }
    setPasswordLoading(true);
    try {
      await usersApi.changeUserPassword(passwordUser.id, values);
      message.success(t("users.passwordChanged"));
      setPasswordUser(null);
    } catch (e) {
      message.error(formatApiError(e, t("users.passwordChangeFailed")));
    } finally {
      setPasswordLoading(false);
    }
  };

  const confirmToggleEnabled = (user: User) => {
    const disabling = user.enabled;
    modal.confirm({
      title: disabling
        ? t("users.disableConfirm", { username: user.username })
        : t("users.enableConfirm", { username: user.username }),
      okText: disabling ? t("common.disable") : t("common.enable"),
      okType: disabling ? "danger" : "primary",
      cancelText: t("common.cancel"),
      onOk: async () => {
        try {
          const updated = await usersApi.setUserEnabled(user.id, {
            enabled: !user.enabled,
          });
          setData((prev) =>
            prev.map((row) => (row.id === updated.id ? updated : row)),
          );
          message.success(
            disabling ? t("users.disabled") : t("users.enabled"),
          );
        } catch (e) {
          message.error(formatApiError(e, t("users.statusChangeFailed")));
        }
      },
    });
  };

  const confirmDelete = (user: User) => {
    modal.confirm({
      title: t("users.deleteConfirm", { username: user.username }),
      okText: t("common.delete"),
      okType: "danger",
      cancelText: t("common.cancel"),
      onOk: async () => {
        try {
          await usersApi.deleteUser(user.id);
          message.success(t("users.deleted"));
          await load();
        } catch (e) {
          message.error(formatApiError(e, t("users.deleteFailed")));
        }
      },
    });
  };

  const columns: ColumnsType<User> = [
    {
      title: t("common.username"),
      dataIndex: "username",
      key: "username",
    },
    {
      title: t("common.status"),
      dataIndex: "enabled",
      key: "enabled",
      render: (enabled: boolean) =>
        enabled ? (
          <Tag color="success">{t("users.active")}</Tag>
        ) : (
          <Tag color="default">{t("users.inactive")}</Tag>
        ),
    },
    {
      title: t("common.created"),
      dataIndex: "created_at",
      key: "created_at",
      render: (v: string) => new Date(v).toLocaleString(),
    },
    {
      title: t("common.actions"),
      key: "actions",
      render: (_, record) => (
        <Space wrap>
          <Button type="link" onClick={() => setPasswordUser(record)}>
            {t("common.password")}
          </Button>
          <Button
            type="link"
            danger={record.enabled}
            disabled={record.protected && record.enabled}
            onClick={() => confirmToggleEnabled(record)}
          >
            {record.enabled ? t("common.disable") : t("common.enable")}
          </Button>
          <Button
            type="link"
            danger
            disabled={record.protected}
            onClick={() => confirmDelete(record)}
          >
            {t("common.delete")}
          </Button>
        </Space>
      ),
    },
  ];

  const onTableChange = (pagination: TablePaginationConfig) => {
    setPage(pagination.current ?? 1);
  };

  return (
    <div className="flex w-full flex-col gap-4">
      <div className="flex flex-row flex-wrap justify-between gap-2">
        <Input.Search
          placeholder={t("users.searchPlaceholder")}
          allowClear
          value={searchInput}
          onChange={(e) => setSearchInput(e.target.value)}
          onSearch={(value) => {
            setSearch(value.trim());
            setPage(1);
          }}
          className="max-w-sm"
        />
        <Button type="primary" onClick={() => setCreateOpen(true)}>
          {t("common.add")}
        </Button>
      </div>

      <PanelTable>
        <Table<User>
          rowKey="id"
          loading={loading}
          columns={columns}
          dataSource={data}
          pagination={{
            current: page,
            pageSize: PAGE_SIZE,
            total,
            showSizeChanger: false,
          }}
          onChange={onTableChange}
          showHeader
          size="middle"
          rowClassName={() => "row-hover"}
        />
      </PanelTable>

      <CreateUserModal
        open={createOpen}
        loading={createLoading}
        onCancel={() => setCreateOpen(false)}
        onSubmit={handleCreate}
      />
      <ChangePasswordModal
        open={passwordUser !== null}
        loading={passwordLoading}
        user={passwordUser}
        onCancel={() => setPasswordUser(null)}
        onSubmit={handleChangePassword}
      />
    </div>
  );
}
