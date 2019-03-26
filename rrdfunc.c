#include <stdlib.h>
#define RRD_EXPORT_DEPRECATED 1
#include <rrd.h>
#include <rrd_format.h>
#include <math.h>
#include "rrdfunc.h"


#define RRD_READONLY    (1<<0)
#define RRD_READWRITE   (1<<1)
#define RRD_CREAT       (1<<2)
#define RRD_READAHEAD   (1<<3)
#define RRD_COPY        (1<<4)
#define RRD_EXCL        (1<<5)
#define RRD_READVALUES  (1<<6)
#define RRD_LOCK        (1<<7)


char *rrdError() {
	char *err = NULL;
	if (rrd_test_error()) {
		// RRD error is local for thread so other gorutine can call some RRD
		// function in the same thread before we use C.GoString. So we need to
		// copy current error before return from C to Go. It need to be freed
		// after C.GoString in Go code.
		err = strdup(rrd_get_error());
		if (err == NULL) {
			abort();
		}
	}
	return err;
}

char *rrdCreate(const char *filename, unsigned long step, time_t start, int argc, const char **argv) {
	rrd_clear_error();
	rrd_create_r(filename, step, start, argc, argv);
	return rrdError();
}

char *rrdUpdate(const char *filename, const char *template, int argc, const char **argv) {
	rrd_clear_error();
	rrd_update_r(filename, template, argc, argv);
	return rrdError();
}

char *rrdGraph(rrd_info_t **ret, int argc, char **argv) {
	rrd_clear_error();
	*ret = rrd_graph_v(argc, argv);
	return rrdError();
}

char *rrdInfo(rrd_info_t **ret, char *filename) {
	rrd_clear_error();
	*ret = rrd_info_r(filename);
	return rrdError();
}

char *rrdFetch(int *ret, char *filename, const char *cf, time_t *start, time_t *end, unsigned long *step, unsigned long *ds_cnt, char ***ds_namv, double **data) {
	rrd_clear_error();
	*ret = rrd_fetch_r(filename, cf, start, end, step, ds_cnt, ds_namv, data);
	return rrdError();
}

char *rrdXport(int *ret, int argc, char **argv, int *xsize, time_t *start, time_t *end, unsigned long *step, unsigned long *col_cnt, char ***legend_v, double **data) {
	rrd_clear_error();
	*ret = rrd_xport(argc, argv, xsize, start, end, step, col_cnt, legend_v, data);
	return rrdError();
}

char *arrayGetCString(char **values, int i) {
	return values[i];
}

typedef struct rrd_dumper_t {
    rrd_file_t *rrd_file;
    rrd_t *rrd;
    off_t rra_base;
    off_t rra_start;
    off_t rra_next;

    long timer;

    int num_rra; // i
    int rra_cur_row; // ix
    int rra_row_cnt; // ii
} rrd_dumper_t;

rrd_dumper_t* init_rrd_dumper(char *filename) {
    rrd_dumper_t *dumper = (rrd_dumper_t*)malloc(sizeof(rrd_dumper_t));
    dumper->rrd = (rrd_t*)malloc(sizeof(rrd_t));

    rrd_init(dumper->rrd);

    dumper->rrd_file = rrd_open(filename, dumper->rrd, RRD_READONLY | RRD_LOCK |
                                               RRD_READAHEAD);
    if (dumper->rrd_file == NULL) {
        close_rrd_dumper(dumper);
        free(dumper);
        return NULL;
    }

    dumper->rra_base = dumper->rrd_file->header_len;
    dumper->rra_next = dumper->rra_base;
    dumper->rra_start = dumper->rra_next;

    return dumper;
}

/*
 * This function changes the pointer of the RRA to the specified definition
 * usually "AVERAGE"
 */
int seek_rrd_dumper_to_def(rrd_dumper_t* dumper, char *rra_definition_name) {
    int i;

    for (i = 0; i < dumper->rrd->stat_head->rra_cnt; i++) {
        dumper->rra_start = dumper->rra_next;
        dumper->rra_next += (dumper->rrd->stat_head->ds_cnt
                             * dumper->rrd->rra_def[i].row_cnt * sizeof(rrd_value_t));

        if(strcmp(dumper->rrd->rra_def[i].cf_nam, rra_definition_name) == 0) {
            dumper->num_rra = i;
            rrd_seek(dumper->rrd_file, (dumper->rra_start + (dumper->rrd->rra_ptr[dumper->num_rra].cur_row + 1)
                    * dumper->rrd->stat_head->ds_cnt
                    * sizeof(rrd_value_t)), SEEK_SET);
            dumper->timer = -(long)(dumper->rrd->rra_def[dumper->num_rra].row_cnt - 1);
            dumper->rra_row_cnt = dumper->rrd->rra_ptr[dumper->num_rra].cur_row;
            dumper->rra_cur_row = 0;
            return 0;
        }
    }
    return 1;
}

rrd_row_dump_t* yield_rrd_dumper_row(rrd_dumper_t* dumper) {
    rrd_value_t val;
    int cur_col;
//ix = 0; ix < rrd.rra_def[i].row_cnt; ix++
    if(dumper->rra_cur_row >= dumper->rrd->rra_def[dumper->num_rra].row_cnt) {
        // reached end of rra
        return NULL;
    }
    dumper->rra_row_cnt++;
    if(dumper->rra_row_cnt >= dumper->rrd->rra_def[dumper->num_rra].row_cnt) {
        rrd_seek(dumper->rrd_file, dumper->rra_start, SEEK_SET);
        dumper->rra_row_cnt = 0;
    }

    rrd_row_dump_t *row = (rrd_row_dump_t*)malloc(sizeof(rrd_row_dump_t));
    
    row->timestamp = (dumper->rrd->live_head->last_up
                        - dumper->rrd->live_head->last_up
                        % (dumper->rrd->rra_def[dumper->num_rra].pdp_cnt * dumper->rrd->stat_head->pdp_step))
                    + (dumper->timer * (long)dumper->rrd->rra_def[dumper->num_rra].pdp_cnt * (long)dumper->rrd->stat_head->pdp_step);
    dumper->timer++;

    row->num_cols = dumper->rrd->stat_head->ds_cnt;
    row->cols = (double*)malloc(sizeof(double)*row->num_cols);
    
    for (cur_col = 0; cur_col < row->num_cols; cur_col++) {
        rrd_read(dumper->rrd_file, (row->cols + cur_col), sizeof(rrd_value_t) * 1);
    }

    dumper->rra_cur_row++;
    return row;
}

void close_rrd_dumper(rrd_dumper_t* dumper) {
    rrd_free(dumper->rrd);
    rrd_close(dumper->rrd_file);
}
