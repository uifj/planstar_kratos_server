import {
  type ActionType,
  PageContainer,
  type ProColumns,
  ProTable,
} from '@ant-design/pro-components';
import { FormattedMessage, useAccess, useIntl } from '@umijs/max';
import { App, Popconfirm, Typography } from 'antd';
import React, { useRef } from 'react';
import { type Refund, services } from '@/services';
import {
  formatCents,
  PAGE_SIZE,
  REFUND_STATUS,
  statusValueEnum,
  useCursorPager,
} from '../shared';

type RefundQuery = {
  keyword?: string;
  status?: string;
};

const RefundTable: React.FC = () => {
  const actionRef = useRef<ActionType>(null);
  const pager = useCursorPager();
  const { canOperate } = useAccess();
  const { message } = App.useApp();
  const intl = useIntl();

  const review = async (record: Refund, decision: 'approve' | 'reject') => {
    try {
      await services.console.ReviewRefund({ id: record.id, decision });
      message.success(
        intl.formatMessage({ id: 'pages.planstar.refunds.reviewSuccess' }),
      );
      actionRef.current?.reload();
    } catch {
      // 失败提示由 requestErrorConfig 统一处理。
    }
  };

  const columns: ProColumns<Refund>[] = [
    {
      title: <FormattedMessage id="pages.planstar.search.keyword" />,
      dataIndex: 'keyword',
      hideInTable: true,
      fieldProps: {
        placeholder: intl.formatMessage({
          id: 'pages.planstar.refunds.keywordPlaceholder',
        }),
      },
    },
    {
      title: 'ID',
      dataIndex: 'id',
      width: 90,
      search: false,
      copyable: true,
    },
    {
      title: <FormattedMessage id="pages.planstar.refunds.refundNo" />,
      dataIndex: 'refundNo',
      search: false,
      width: 200,
      ellipsis: true,
      copyable: true,
    },
    {
      title: <FormattedMessage id="pages.planstar.refunds.orderNo" />,
      dataIndex: 'orderNo',
      search: false,
      width: 180,
      ellipsis: true,
      render: (_, record) => record.orderNo || '-',
    },
    {
      title: <FormattedMessage id="pages.planstar.userId" />,
      dataIndex: 'userId',
      search: false,
      width: 110,
      render: (_, record) => record.userId ?? '-',
    },
    {
      title: <FormattedMessage id="pages.planstar.refunds.reasonCode" />,
      dataIndex: 'reasonCode',
      search: false,
      width: 130,
      render: (_, record) => record.reasonCode || '-',
    },
    {
      title: <FormattedMessage id="pages.planstar.refunds.amount" />,
      dataIndex: 'refundableAmount',
      search: false,
      width: 130,
      align: 'right',
      render: (_, record) => formatCents(record.refundableAmount),
    },
    {
      title: <FormattedMessage id="pages.planstar.refunds.status" />,
      dataIndex: 'status',
      width: 120,
      valueEnum: statusValueEnum(intl, REFUND_STATUS),
    },
    {
      title: <FormattedMessage id="pages.planstar.createdAt" />,
      dataIndex: 'createdAt',
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
            width: 140,
            search: false,
            render: (_, record) => {
              const reviewable =
                record.status === 'applied' || record.status === 'processing';
              if (!reviewable) return ['-'];
              return [
                <Popconfirm
                  key="approve"
                  title={intl.formatMessage({
                    id: 'pages.planstar.refunds.approveConfirm.title',
                  })}
                  description={intl.formatMessage({
                    id: 'pages.planstar.refunds.approveConfirm.description',
                  })}
                  onConfirm={() => review(record, 'approve')}
                >
                  <a>
                    {intl.formatMessage({
                      id: 'pages.planstar.refunds.approve',
                    })}
                  </a>
                </Popconfirm>,
                <Popconfirm
                  key="reject"
                  title={intl.formatMessage({
                    id: 'pages.planstar.refunds.rejectConfirm.title',
                  })}
                  description={intl.formatMessage({
                    id: 'pages.planstar.refunds.rejectConfirm.description',
                  })}
                  onConfirm={() => review(record, 'reject')}
                >
                  <a>
                    {intl.formatMessage({
                      id: 'pages.planstar.refunds.reject',
                    })}
                  </a>
                </Popconfirm>,
              ];
            },
          },
        ] as ProColumns<Refund>[])
      : []),
  ];

  return (
    <PageContainer
      title={intl.formatMessage({ id: 'pages.planstar.refunds.title' })}
      content={
        <Typography.Text type="secondary">
          <FormattedMessage id="pages.planstar.refunds.description" />
        </Typography.Text>
      }
    >
      <ProTable<Refund, RefundQuery>
        actionRef={actionRef}
        rowKey="id"
        cardBordered
        columns={columns}
        search={{ labelWidth: 'auto' }}
        options={{ density: false, fullScreen: false, setting: false }}
        pagination={{ pageSize: PAGE_SIZE, showSizeChanger: false }}
        request={async (params) => {
          const keyword = params.keyword?.trim() || undefined;
          const status = params.status || undefined;
          const current = params.current ?? 1;
          const pageToken = pager.pageToken(
            current,
            `${keyword ?? ''}|${status ?? ''}`,
          );
          const res = await services.console.ListRefunds({
            keyword,
            status,
            pageSize: PAGE_SIZE,
            pageToken,
          });
          const data = res.refunds ?? [];
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

export default RefundTable;
