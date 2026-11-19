import { type AuditLog, services } from '@/services';
import {
  PageContainer,
  type ActionType,
  type ProColumns,
  ProTable,
} from '@ant-design/pro-components';
import { FormattedMessage, useIntl } from '@umijs/max';
import { Typography } from 'antd';
import React, { useRef } from 'react';
import { PAGE_SIZE, useCursorPager } from '../planstar/shared';

type AuditQuery = {
  keyword?: string;
  action?: string;
  result?: string;
};

const AuditLogTable: React.FC = () => {
  const actionRef = useRef<ActionType>(null);
  const pager = useCursorPager();
  const intl = useIntl();

  const columns: ProColumns<AuditLog>[] = [
    {
      title: <FormattedMessage id="pages.audit.keyword" />,
      dataIndex: 'keyword',
      hideInTable: true,
      fieldProps: {
        placeholder: intl.formatMessage({
          id: 'pages.audit.keywordPlaceholder',
        }),
      },
    },
    {
      title: <FormattedMessage id="pages.audit.action" />,
      dataIndex: 'action',
      width: 160,
      fieldProps: {
        placeholder: intl.formatMessage({
          id: 'pages.audit.actionPlaceholder',
        }),
      },
      render: (_, record) => (
        <Typography.Text code>{record.action || '-'}</Typography.Text>
      ),
    },
    {
      title: <FormattedMessage id="pages.audit.result" />,
      dataIndex: 'result',
      width: 100,
      // valueEnum 的 text 必须是纯字符串：搜索下拉的默认 filterOption 会对
      // label 做字符串匹配，React 节点会被匹配成 "[object Object]" 而永远搜不到。
      valueEnum: {
        success: {
          text: intl.formatMessage({ id: 'pages.audit.result.success' }),
          status: 'Success',
        },
        failure: {
          text: intl.formatMessage({ id: 'pages.audit.result.failure' }),
          status: 'Error',
        },
      },
    },
    {
      title: 'ID',
      dataIndex: 'id',
      width: 90,
      search: false,
    },
    {
      title: <FormattedMessage id="pages.audit.actor" />,
      dataIndex: 'actor',
      search: false,
      width: 160,
      ellipsis: true,
      render: (_, record) => record.actor || '-',
    },
    {
      title: <FormattedMessage id="pages.audit.target" />,
      dataIndex: 'target',
      search: false,
      width: 120,
      render: (_, record) => record.target || '-',
    },
    {
      title: <FormattedMessage id="pages.audit.detail" />,
      dataIndex: 'detail',
      search: false,
      width: 260,
      ellipsis: true,
      render: (_, record) => record.detail || '-',
    },
    {
      title: <FormattedMessage id="pages.audit.remoteAddr" />,
      dataIndex: 'remoteAddr',
      search: false,
      width: 150,
      render: (_, record) => record.remoteAddr || '-',
    },
    {
      title: <FormattedMessage id="pages.audit.createdAt" />,
      dataIndex: 'createdAt',
      valueType: 'dateTime',
      width: 200,
      search: false,
    },
  ];

  return (
    <PageContainer
      title={intl.formatMessage({ id: 'pages.audit.title' })}
      content={
        <Typography.Text type="secondary">
          <FormattedMessage id="pages.audit.description" />
        </Typography.Text>
      }
    >
      <ProTable<AuditLog, AuditQuery>
        actionRef={actionRef}
        rowKey="id"
        cardBordered
        columns={columns}
        search={{ labelWidth: 'auto' }}
        options={{ density: false, fullScreen: false, setting: false }}
        pagination={{ pageSize: PAGE_SIZE, showSizeChanger: false }}
        request={async (params) => {
          const keyword = params.keyword?.trim() || undefined;
          const action = params.action?.trim() || undefined;
          const result = params.result || undefined;
          const current = params.current ?? 1;
          const pageToken = pager.pageToken(
            current,
            `${keyword ?? ''}|${action ?? ''}|${result ?? ''}`,
          );
          const res = await services.console.ListAuditLogs({
            keyword,
            action,
            result,
            pageSize: PAGE_SIZE,
            pageToken,
          });
          const data = res.logs ?? [];
          return {
            data,
            success: true,
            total: pager.total(current, data.length, res.nextPageToken),
          };
        }}
      />
    </PageContainer>
  );
};

export default AuditLogTable;