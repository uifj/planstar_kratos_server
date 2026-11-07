import { type Project, services } from '@/services';
import {
  PageContainer,
  type ActionType,
  type ProColumns,
  ProTable,
} from '@ant-design/pro-components';
import { FormattedMessage, useAccess, useIntl } from '@umijs/max';
import { Typography } from 'antd';
import React, { useRef } from 'react';
import {
  PAGE_SIZE,
  formatCents,
  formatPermille,
  optionalIntFilter,
  useCursorPager,
} from '../shared';
import ReviewForm from './ReviewForm';

type ProjectQuery = {
  keyword?: string;
  publishStatus?: string;
};

const ProjectTable: React.FC = () => {
  const actionRef = useRef<ActionType>(null);
  const pager = useCursorPager();
  const { canOperate } = useAccess();
  const intl = useIntl();

  const columns: ProColumns<Project>[] = [
    {
      title: <FormattedMessage id="pages.planstar.search.keyword" />,
      dataIndex: 'keyword',
      hideInTable: true,
      fieldProps: {
        placeholder: intl.formatMessage({
          id: 'pages.planstar.projects.keywordPlaceholder',
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
      title: <FormattedMessage id="pages.planstar.projects.titleColumn" />,
      dataIndex: 'title',
      search: false,
      ellipsis: true,
      render: (_, record) => (
        <Typography.Text strong>{record.title || '-'}</Typography.Text>
      ),
    },
    {
      title: <FormattedMessage id="pages.planstar.projects.category" />,
      dataIndex: 'categoryName',
      search: false,
      width: 140,
      render: (_, record) => record.categoryName || '-',
    },
    {
      title: <FormattedMessage id="pages.planstar.projects.publishStatus" />,
      dataIndex: 'publishStatus',
      width: 110,
      valueEnum: {
        0: {
          text: <FormattedMessage id="pages.planstar.projects.publish.draft" />,
          status: 'Default',
        },
        1: {
          text: (
            <FormattedMessage id="pages.planstar.projects.publish.published" />
          ),
          status: 'Success',
        },
        2: {
          text: (
            <FormattedMessage id="pages.planstar.projects.publish.reviewing" />
          ),
          status: 'Processing',
        },
        3: {
          text: (
            <FormattedMessage id="pages.planstar.projects.publish.rejected" />
          ),
          status: 'Error',
        },
      },
    },
    {
      title: <FormattedMessage id="pages.planstar.projects.fundingStatus" />,
      dataIndex: 'fundingStatus',
      search: false,
      width: 110,
      valueEnum: {
        preheating: {
          text: (
            <FormattedMessage id="pages.planstar.projects.funding.preheating" />
          ),
          status: 'Default',
        },
        ongoing: {
          text: <FormattedMessage id="pages.planstar.projects.funding.ongoing" />,
          status: 'Processing',
        },
        achieved: {
          text: (
            <FormattedMessage id="pages.planstar.projects.funding.achieved" />
          ),
          status: 'Success',
        },
        delivered: {
          text: (
            <FormattedMessage id="pages.planstar.projects.funding.delivered" />
          ),
          status: 'Success',
        },
        failed: {
          text: <FormattedMessage id="pages.planstar.projects.funding.failed" />,
          status: 'Error',
        },
      },
    },
    {
      title: <FormattedMessage id="pages.planstar.projects.goalAmount" />,
      dataIndex: 'goalAmount',
      search: false,
      width: 130,
      align: 'right',
      render: (_, record) => formatCents(record.goalAmount),
    },
    {
      title: <FormattedMessage id="pages.planstar.projects.achievement" />,
      dataIndex: 'achievementPermille',
      search: false,
      width: 100,
      align: 'right',
      render: (_, record) => formatPermille(record.achievementPermille),
    },
    {
      title: <FormattedMessage id="pages.planstar.projects.supporters" />,
      dataIndex: 'supporterCount',
      search: false,
      width: 100,
      align: 'right',
      render: (_, record) => record.supporterCount ?? 0,
    },
    {
      title: <FormattedMessage id="pages.planstar.projects.createdAt" />,
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
            width: 100,
            search: false,
            render: (_, record) => [
              <ReviewForm
                key="review"
                record={record}
                trigger={
                  <a>
                    <FormattedMessage id="pages.planstar.projects.review" />
                  </a>
                }
                onOk={() => actionRef.current?.reload()}
              />,
            ],
          },
        ] as ProColumns<Project>[])
      : []),
  ];

  return (
    <PageContainer
      title={intl.formatMessage({ id: 'pages.planstar.projects.title' })}
      content={
        <Typography.Text type="secondary">
          <FormattedMessage id="pages.planstar.projects.description" />
        </Typography.Text>
      }
    >
      <ProTable<Project, ProjectQuery>
        actionRef={actionRef}
        rowKey="id"
        cardBordered
        columns={columns}
        search={{ labelWidth: 'auto' }}
        options={{ density: false, fullScreen: false, setting: false }}
        pagination={{ pageSize: PAGE_SIZE, showSizeChanger: false }}
        request={async (params) => {
          const keyword = params.keyword?.trim() || undefined;
          const publishStatus =
            params.publishStatus === undefined || params.publishStatus === ''
              ? undefined
              : Number(params.publishStatus);
          const current = params.current ?? 1;
          const pageToken = pager.pageToken(
            current,
            `${keyword ?? ''}|${publishStatus ?? ''}`,
          );
          const res = await services.console.ListProjects({
            keyword,
            publishStatus: optionalIntFilter(publishStatus),
            pageSize: PAGE_SIZE,
            pageToken,
          });
          const data = res.projects ?? [];
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

export default ProjectTable;
