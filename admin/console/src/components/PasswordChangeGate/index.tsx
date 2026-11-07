import { services } from "@/services";
import { history, useIntl, useModel } from "@umijs/max";
import { App, Button, Form, Input, Modal, Typography } from "antd";
import React, { useState } from "react";

type PasswordForm = {
  oldPassword: string;
  newPassword: string;
  confirmPassword: string;
};

/**
 * 首次登录（或密码被重置）时必须先改密。服务端签发的 Cookie 在改密前只允许
 * 读取资料、改密和退出，所以这里用不可关闭的 Modal 阻断其余操作。
 */
const PasswordChangeGate: React.FC = () => {
  const { initialState, setInitialState } = useModel("@@initialState");
  const { message } = App.useApp();
  const intl = useIntl();
  const [form] = Form.useForm<PasswordForm>();
  const [submitting, setSubmitting] = useState(false);
  const [loggingOut, setLoggingOut] = useState(false);

  const mustChange = Boolean(
    initialState?.currentUser?.mustChangePassword,
  );
  if (!mustChange) {
    return null;
  }

  const submit = async () => {
    let values: PasswordForm;
    try {
      values = await form.validateFields();
    } catch {
      return;
    }
    setSubmitting(true);
    try {
      await services.admin.ChangePassword({
        oldPassword: values.oldPassword,
        newPassword: values.newPassword,
      });
      message.success(
        intl.formatMessage({ id: "pages.passwordChange.success" }),
      );
      window.location.reload();
    } catch {
      // 旧密码错误等失败由 requestErrorConfig 统一提示。
    } finally {
      setSubmitting(false);
    }
  };

  const logout = async () => {
    setLoggingOut(true);
    try {
      await services.admin.Logout({});
    } catch {
      // 即使会话已失效也继续回到登录页。
    } finally {
      setInitialState((state) => ({ ...state, currentUser: undefined }));
      history.replace("/user/login");
      setLoggingOut(false);
    }
  };

  return (
    <Modal
      open
      centered
      title={intl.formatMessage({ id: "pages.passwordChange.title" })}
      closable={false}
      maskClosable={false}
      keyboard={false}
      footer={[
        <Button key="logout" onClick={logout} loading={loggingOut}>
          {intl.formatMessage({ id: "pages.passwordChange.logout" })}
        </Button>,
        <Button
          key="submit"
          type="primary"
          onClick={submit}
          loading={submitting}
        >
          {intl.formatMessage({ id: "pages.passwordChange.submit" })}
        </Button>,
      ]}
    >
      <Typography.Paragraph type="secondary">
        {intl.formatMessage({ id: "pages.passwordChange.description" })}
      </Typography.Paragraph>
      <Form<PasswordForm> form={form} layout="vertical" requiredMark={false}>
        <Form.Item
          name="oldPassword"
          label={intl.formatMessage({ id: "pages.passwordChange.old" })}
          rules={[
            {
              required: true,
              message: intl.formatMessage({
                id: "pages.passwordChange.oldRequired",
              }),
            },
          ]}
        >
          <Input.Password
            autoComplete="current-password"
            placeholder={intl.formatMessage({
              id: "pages.passwordChange.oldPlaceholder",
            })}
          />
        </Form.Item>
        <Form.Item
          name="newPassword"
          label={intl.formatMessage({ id: "pages.passwordChange.new" })}
          rules={[
            {
              required: true,
              message: intl.formatMessage({
                id: "pages.passwordChange.newRequired",
              }),
            },
            {
              min: 8,
              message: intl.formatMessage({
                id: "pages.passwordChange.minLength",
              }),
            },
          ]}
        >
          <Input.Password
            autoComplete="new-password"
            placeholder={intl.formatMessage({
              id: "pages.passwordChange.newPlaceholder",
            })}
          />
        </Form.Item>
        <Form.Item
          name="confirmPassword"
          label={intl.formatMessage({ id: "pages.passwordChange.confirm" })}
          dependencies={["newPassword"]}
          rules={[
            {
              required: true,
              message: intl.formatMessage({
                id: "pages.passwordChange.confirmRequired",
              }),
            },
            ({ getFieldValue }) => ({
              validator(_, value) {
                if (!value || getFieldValue("newPassword") === value) {
                  return Promise.resolve();
                }
                return Promise.reject(
                  new Error(
                    intl.formatMessage({
                      id: "pages.passwordChange.mismatch",
                    }),
                  ),
                );
              },
            }),
          ]}
        >
          <Input.Password
            autoComplete="new-password"
            placeholder={intl.formatMessage({
              id: "pages.passwordChange.confirmPlaceholder",
            })}
          />
        </Form.Item>
      </Form>
    </Modal>
  );
};

export default PasswordChangeGate;
