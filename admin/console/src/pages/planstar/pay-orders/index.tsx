import { type PayOrder, services } from '@/services';
import {
  PageContainer,
  type ActionType,
  type ProColumns,
  ProTable,
} from '@ant-design/pro-components';
import { FormattedMessage, useIntl } from '@umijs/max';
import { Typography } from 'antd';
import React, { useRef } from 'react';
import {
  PAY_ORDER_STATUS,
  PAGE_SIZE,
  formatCents,
  statusValueEnum,
  useCursorPager,
} from '../shared';

type PayOrderQuery = {
  keyword?: string;
  status?: string;
};

const PayOrderTable: React.FC = () => {
  const actionRef = useRef<ActionType>(null);
  const pager = useCursorPager();
  const intl = useIntl();

  const columns: ProColumns<PayOrder>[] = [
    {
      title: <FormattedMessage id="pages.planstar.search.keyword" />,
      dataIndex: 'keyword',
      hideInTable: true,
      fieldProps: {
        placeholder: intl.formatMessage({
          id: 'pages.planstar.payOrders.keywordPlaceholder',
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
      title: <FormattedMessage id="pages.planstar.payOrders.payNo" />,
      dataIndex: 'payNo',
      search: false,
      width: 200,
      ellipsis: true,
      copyable: true,
    },
    {
      title: <FormattedMessage id="pages.planstar.payOrders.orderNo" />,
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
      title: <FormattedMessage id="pages.planstar.payOrders.channel" />,
      dataIndex: 'channel',
      search: false,
      width: 110,
      render: (_, record) => record.channel || '-',
    },
    {
      title: <FormattedMessage id="pages.planstar.payOrders.amount" />,
      dataIndex: 'amount',
      search: false,
      width: 130,
      align: 'right',
      render: (_, record) => formatCents(record.amount),
    },
    {
      title: <FormattedMessage id="pages.planstar.payOrders.status" />,
      dataIndex: 'status',
      width: 120,
      valueEnum: statusValueEnum(intl, PAY_ORDER_STATUS),
    },
    {
      title: <FormattedMessage id="pages.planstar.payOrders.paidAt" />,
      dataIndex: 'paidAt',
      valueType: 'dateTime',
      width: 200,
      search: false,
    },
    {
      title: <FormattedMessage id="pages.planstar.createdAt" />,
      dataIndex: 'createdAt',
      valueType: 'dateTime',
      width: 200,
      search: false,
    },
  ];

  return (
    <PageContainer
      title={intl.formatMessage({ id: 'pages.planstar.payOrders.title' })}
      content={
        <Typography.Text type="secondary">
          <FormattedMessage id="pages.planstar.payOrders.description" />
        </Typography.Text>
      }
    >
      <ProTable<PayOrder, PayOrderQuery>
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
          const res = await services.console.ListPayOrders({
            keyword,
            status,
            pageSize: PAGE_SIZE,
            pageToken,
          });
          const data = res.payOrders ?? [];
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

export default PayOrderTable;