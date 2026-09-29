"use client";

import { Alert, Button, Card, Form, Input, Typography } from "antd";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { ApiError } from "@/api/base";
import { OktopusIcon } from "@/assets/components/oktopus/OktopusIcon";
import { OktopusLoading } from "@/assets/components/oktopus/OktopusLoading";
import { useAuth } from "@/contexts/AuthContext";

type LoginForm = {
  username: string;
  password: string;
};

export default function LoginPage() {
  const { user, loading, login } = useAuth();
  const router = useRouter();
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    if (!loading && user) {
      router.replace("/manage");
    }
  }, [loading, user, router]);

  const onFinish = async (values: LoginForm) => {
    setError(null);
    setSubmitting(true);
    try {
      await login(values);
      router.replace("/manage");
    } catch (e) {
      if (e instanceof ApiError) {
        setError(e.message);
      } else {
        setError("Не удалось войти");
      }
    } finally {
      setSubmitting(false);
    }
  };

  if (loading || user) {
    return (
      <main className="grid-bg flex min-h-full flex-1 items-center justify-center p-4">
        <OktopusLoading size="xl" />
      </main>
    );
  }

  return (
    <main className="grid-bg flex min-h-full flex-1 items-center justify-center p-4">
      <Card className="login-panel w-full max-w-md !rounded-xl">
        <div className="flex flex-col gap-6">
          <div className="flex flex-col items-center gap-2">
            <span className="grid h-9 w-9 place-items-center rounded-md border border-teal-400/25 bg-teal-400/10">
              <OktopusIcon size="lg" title="Oktopus" />
            </span>
            <Typography.Title
              level={4}
              className="!mb-0 !text-zinc-200"
            >
              Вход в Oktopus
            </Typography.Title>
          </div>
          {error ? <Alert type="error" message={error} showIcon /> : null}
          <Form<LoginForm> layout="vertical" onFinish={onFinish}>
            <Form.Item
              name="username"
              label="Логин"
              rules={[{ required: true, message: "Введите логин" }]}
            >
              <Input autoComplete="username" />
            </Form.Item>
            <Form.Item
              name="password"
              label="Пароль"
              rules={[{ required: true, message: "Введите пароль" }]}
            >
              <Input.Password autoComplete="current-password" />
            </Form.Item>
            <Button
              type="primary"
              htmlType="submit"
              block
              loading={submitting}
            >
              Войти
            </Button>
          </Form>
        </div>
      </Card>
    </main>
  );
}
