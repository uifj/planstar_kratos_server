import { type Report, services } from '@/services';
import {
  PageContainer,
  type ActionType,
  type ProColumns,
  ProTable,
} from '@ant-design/pro-components';
import { FormattedMessage, useAccess, useIntl } from '@umijs/max';
import { Typography } from 'antd';
import React, { useRef } from 'react';
import { PAGE_SIZE, useCursorPager } from '../shared';
import HandleForm from './HandleForm';

type ReportQuery = {
  status?: string;
};

const ReportTable: React.FC = () => {
  const actionRef = useRef<ActionType>(null);
  const pager = useCursorPager();
  const { canOperate } = useAccess();
  const intl = useIntl();

  const columns: ProColumns<Report>[] = [
    {
      title: <FormattedMessage id="pages.planstar.reports.id" />,
      dataIndex: 'id',
      width: 100,
      search: false,
      copyable: true,
    },
    {
      title: <FormattedMessage id="pages.planstar.reports.targetType" />,
      dataIndex: 'targetType',
      search: false,
      width: 110,
      render: (_, record) => record.targetType || '-',
    },
    {
      title: <FormattedMessage id="pages.planstar.reports.targetId" />,
      dataIndex: 'targetId',
      search: false,
      width: 110,
      render: (_, record) => record.targetId ?? '-',
    },
    {
      title: <FormattedMessage id="pages.planstar.reports.reporter" />,
      dataIndex: 'userId',
      search: false,
      width: 110,
      render: (_, record) => record.userId ?? '-',
    },
    {
      title: <FormattedMessage id="pages.planstar.reports.reasonCode" />,
      dataIndex: 'reasonCode',
      search: false,
      width: 130,
      render: (_, record) =>
        record.reasonCode
          ? intl.formatMessage({
              id: `pages.planstar.reports.reason.${record.reasonCode}`,
              defaultMessage: record.reasonCode,
            })
          : '-',
    },
    {
      title: <FormattedMessage id="pages.planstar.reports.reportDescription" />,
      dataIndex: 'description',
      search: false,
      ellipsis: true,
      render: (_, record) => record.description || '-',
    },
    {
      title: <FormattedMessage id="pages.planstar.reports.status" />,
      dataIndex: 'status',
      width: 110,
      valueEnum: {
        pending: {
          text: <FormattedMessage id="pages.planstar.reports.status.pending" />,
          status: 'Processing',
        },
        accepted: {
          text: <FormattedMessage id="pages.planstar.reports.status.accepted" />,
          status: 'Success',
        },
        rejected: {
          text: <FormattedMessage id="pages.planstar.reports.status.rejected" />,
          status: 'Default',
        },
      },
    },
    {
      title: <FormattedMessage id="pages.planstar.reports.handleTime" />,
      dataIndex: 'handleTime',
      valueType: 'dateTime',
      width: 200,
      search: false,
    },
    ...(canOperate
      ? ([
          {
            title: <FormattedMessage id="pages.planstar.operation" />,
            dataIndex: 'option',
            valueType: 'option',
            width: 100,
            search: false,
            render: (_, record) => [
              <HandleForm
                key="handle"
                record={record}
                trigger={
                  <a>
                    <FormattedMessage id="pages.planstar.reports.handle" />
                  </a>
                }
                onOk={() => actionRef.current?.reload()}
              />,
            ],
          },
        ] as ProColumns<Report>[])
      : []),
  ];

  return (
    <PageContainer
      title={intl.formatMessage({ id: 'pages.planstar.reports.title' })}
      content={
        <Typography.Text type="secondary">
          <FormattedMessage id="pages.planstar.reports.description" />
        </Typography.Text>
      }
    >
      <ProTable<Report, ReportQuery>
        actionRef={actionRef}
        rowKey="id"
        cardBordered
        columns={columns}
        search={{ labelWidth: 'auto' }}
        options={{ density: false, fullScreen: false, setting: false }}
        pagination={{ pageSize: PAGE_SIZE, showSizeChanger: false }}
        request={async (params) => {
          const status = params.status || undefined;
          const current = params.current ?? 1;
          const pageToken = pager.pageToken(current, status ?? '');
          const res = await services.console.ListReports({
            status,
            pageSize: PAGE_SIZE,
            pageToken,
          });
          const data = res.reports ?? [];
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

export default ReportTable;
