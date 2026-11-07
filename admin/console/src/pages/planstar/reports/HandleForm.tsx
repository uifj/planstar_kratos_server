import { type Report, services } from '@/services';
import {
  ModalForm,
  ProFormRadio,
  ProFormTextArea,
} from '@ant-design/pro-components';
import { FormattedMessage, useIntl } from '@umijs/max';
import { App } from 'antd';
import type { FC, ReactElement } from 'react';

type HandleValues = {
  status: 'accepted' | 'rejected';
  note?: string;
};

export type HandleFormProps = {
  record: Report;
  trigger: ReactElement;
  onOk?: () => void;
};

const HandleForm: FC<HandleFormProps> = ({ record, trigger, onOk }) => {
  const intl = useIntl();
  const { message } = App.useApp();

  const handleFinish = async (values: HandleValues) => {
    try {
      await services.console.HandleReport({
        id: record.id,
        status: values.status,
        note: values.note?.trim() || undefined,
      });
      message.success(
        intl.formatMessage({ id: 'pages.planstar.reports.handle.success' }),
      );
      onOk?.();
      return true;
    } catch {
      // 失败提示由 requestErrorConfig 统一处理。
      return false;
    }
  };

  return (
    <ModalForm<HandleValues>
      title={intl.formatMessage(
        { id: 'pages.planstar.reports.handle.title' },
        { id: record.id },
      )}
      trigger={trigger}
      width={480}
      modalProps={{ destroyOnHidden: true }}
      onFinish={handleFinish}
      initialValues={{ status: 'accepted' }}
    >
      <ProFormRadio.Group
        name="status"
        label={intl.formatMessage({
          id: 'pages.planstar.reports.handle.status',
        })}
        options={[
          {
            label: <FormattedMessage id="pages.planstar.reports.handle.accept" />,
            value: 'accepted',
          },
          {
            label: <FormattedMessage id="pages.planstar.reports.handle.reject" />,
            value: 'rejected',
          },
        ]}
        rules={[
          {
            required: true,
            message: (
              <FormattedMessage id="pages.planstar.reports.handle.statusRequired" />
            ),
          },
        ]}
      />
      <ProFormTextArea
        name="note"
        label={intl.formatMessage({ id: 'pages.planstar.reports.handle.note' })}
        placeholder={intl.formatMessage({
          id: 'pages.planstar.reports.handle.notePlaceholder',
        })}
        fieldProps={{ rows: 4, maxLength: 500, showCount: true }}
      />
    </ModalForm>
  );
};

export default HandleForm;
