import { type Project, asServiceError, services } from '@/services';
import {
  ModalForm,
  ProFormRadio,
} from '@ant-design/pro-components';
import { FormattedMessage, useIntl } from '@umijs/max';
import { App } from 'antd';
import type { FC, ReactElement } from 'react';

type ReviewValues = {
  decision: 'approve' | 'reject';
};

export type ReviewFormProps = {
  record: Project;
  trigger: ReactElement;
  onOk?: () => void;
};

const ReviewForm: FC<ReviewFormProps> = ({ record, trigger, onOk }) => {
  const intl = useIntl();
  const { message } = App.useApp();

  const handleFinish = async (values: ReviewValues) => {
    try {
      await services.console.ReviewProject({
        id: record.id,
        decision: values.decision,
      });
      message.success(
        intl.formatMessage({ id: 'pages.planstar.projects.review.success' }),
      );
      onOk?.();
      return true;
    } catch (error) {
      // 黑名单命中时后端返回 400，message 已包含规则和命中关键词。该请求在
      // requestHandler 中跳过了全局提示，这里负责展示一次。
      const reason = asServiceError(error)?.message;
      message.error(
        reason ||
          intl.formatMessage({ id: 'pages.planstar.projects.review.failed' }),
      );
      return false;
    }
  };

  return (
    <ModalForm<ReviewValues>
      title={intl.formatMessage(
        { id: 'pages.planstar.projects.review.title' },
        { id: record.id },
      )}
      trigger={trigger}
      width={480}
      modalProps={{ destroyOnHidden: true }}
      onFinish={handleFinish}
      initialValues={{ decision: 'approve' }}
    >
      <ProFormRadio.Group
        name="decision"
        label={intl.formatMessage({
          id: 'pages.planstar.projects.review.decision',
        })}
        options={[
          {
            label: (
              <FormattedMessage id="pages.planstar.projects.review.approve" />
            ),
            value: 'approve',
          },
          {
            label: (
              <FormattedMessage id="pages.planstar.projects.review.reject" />
            ),
            value: 'reject',
          },
        ]}
        rules={[
          {
            required: true,
            message: (
              <FormattedMessage id="pages.planstar.projects.review.decisionRequired" />
            ),
          },
        ]}
      />
    </ModalForm>
  );
};

export default ReviewForm;
