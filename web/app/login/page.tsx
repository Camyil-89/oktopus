"use client";

import { Alert, Button, Card, Form, Input, Typography } from "antd";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { AppVersion } from "@/assets/components/AppVersion";
import { LanguageSelect } from "@/assets/components/LanguageSelect";
import { OktopusIcon } from "@/assets/components/oktopus/OktopusIcon";
import { useApiErrorMessage, useTranslation } from "@/contexts/LocaleContext";
import { OktopusLoading } from "@/assets/components/oktopus/OktopusLoading";
import { useAuth } from "@/contexts/AuthContext";

type LoginForm = {
  username: string;
  password: string;
};

export default function LoginPage() {
  const { user, loading, login } = useAuth();
  const formatApiError = useApiErrorMessage();
  const { t } = useTranslation();
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
      setError(formatApiError(e, t("errors.login_failed")));
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
    <main className="grid-bg relative flex min-h-full flex-1 items-center justify-center p-4">
      <div className="absolute right-4 top-4">
        <LanguageSelect />
      </div>
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
              {t("login.title")}
            </Typography.Title>
          </div>
          {error ? <Alert type="error" message={error} showIcon /> : null}
          <Form<LoginForm> layout="vertical" onFinish={onFinish}>
            <Form.Item
              name="username"
              label={t("common.username")}
              rules={[{ required: true, message: t("common.enterUsername") }]}
            >
              <Input autoComplete="username" />
            </Form.Item>
            <Form.Item
              name="password"
              label={t("common.password")}
              rules={[{ required: true, message: t("common.enterPassword") }]}
            >
              <Input.Password autoComplete="current-password" />
            </Form.Item>
            <Button
              type="primary"
              htmlType="submit"
              block
              loading={submitting}
            >
              {t("login.submit")}
            </Button>
          </Form>
        </div>
      </Card>
      <AppVersion
        className="absolute bottom-4 left-0 right-0 text-center font-mono text-[10px] tabular-nums text-zinc-600"
      />
    </main>
  );
}
