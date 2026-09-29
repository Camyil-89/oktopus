"use client";

import { ApiError } from "@/api/base";
import * as usersApi from "@/api/users";
import { PanelTable } from "@/assets/components/PanelTable";
import {
  ChangePasswordModal,
  CreateUserModal,
} from "@/assets/modals/UserFormModals";
import type { User } from "@/types/user";
import { App, Button, Input, Space, Table, Tag } from "antd";
import type { ColumnsType, TablePaginationConfig } from "antd/es/table";
import { useCallback, useEffect, useState } from "react";

const PAGE_SIZE = 20;

export default function ManageUsersPage() {
  const { message, modal } = App.useApp();
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
      const msg =
        e instanceof ApiError ? e.message : "Не удалось загрузить список";
      message.error(msg);
    } finally {
      setLoading(false);
    }
  }, [message, page, search]);

  useEffect(() => {
    void load();
  }, [load]);

  const handleCreate = async (values: { username: string; password: string }) => {
    setCreateLoading(true);
    try {
      await usersApi.createUser(values);
      message.success("Пользователь создан");
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
      const msg =
        e instanceof ApiError ? e.message : "Не удалось создать пользователя";
      message.error(msg);
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
      message.success("Пароль изменён");
      setPasswordUser(null);
    } catch (e) {
      const msg =
        e instanceof ApiError ? e.message : "Не удалось сменить пароль";
      message.error(msg);
    } finally {
      setPasswordLoading(false);
    }
  };

  const confirmToggleEnabled = (user: User) => {
    const disabling = user.enabled;
    modal.confirm({
      title: disabling
        ? `Отключить «${user.username}»?`
        : `Включить «${user.username}»?`,
      okText: disabling ? "Отключить" : "Включить",
      okType: disabling ? "danger" : "primary",
      cancelText: "Отмена",
      onOk: async () => {
        try {
          const updated = await usersApi.setUserEnabled(user.id, {
            enabled: !user.enabled,
          });
          setData((prev) =>
            prev.map((row) => (row.id === updated.id ? updated : row)),
          );
          message.success(disabling ? "Пользователь отключён" : "Пользователь включён");
        } catch (e) {
          const msg =
            e instanceof ApiError ? e.message : "Не удалось изменить статус";
          message.error(msg);
        }
      },
    });
  };

  const confirmDelete = (user: User) => {
    modal.confirm({
      title: `Удалить «${user.username}»?`,
      okText: "Удалить",
      okType: "danger",
      cancelText: "Отмена",
      onOk: async () => {
        try {
          await usersApi.deleteUser(user.id);
          message.success("Пользователь удалён");
          await load();
        } catch (e) {
          const msg =
            e instanceof ApiError ? e.message : "Не удалось удалить";
          message.error(msg);
        }
      },
    });
  };

  const columns: ColumnsType<User> = [
    {
      title: "Логин",
      dataIndex: "username",
      key: "username",
    },
    {
      title: "Статус",
      dataIndex: "enabled",
      key: "enabled",
      render: (enabled: boolean) =>
        enabled ? (
          <Tag color="success">Активен</Tag>
        ) : (
          <Tag color="default">Отключён</Tag>
        ),
    },
    {
      title: "Создан",
      dataIndex: "created_at",
      key: "created_at",
      render: (v: string) => new Date(v).toLocaleString(),
    },
    {
      title: "Действия",
      key: "actions",
      render: (_, record) => (
        <Space wrap>
          <Button type="link" onClick={() => setPasswordUser(record)}>
            Пароль
          </Button>
          <Button
            type="link"
            danger={record.enabled}
            disabled={record.protected && record.enabled}
            onClick={() => confirmToggleEnabled(record)}
          >
            {record.enabled ? "Отключить" : "Включить"}
          </Button>
          <Button
            type="link"
            danger
            disabled={record.protected}
            onClick={() => confirmDelete(record)}
          >
            Удалить
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
          placeholder="Поиск по логину"
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
          Добавить
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
